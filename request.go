// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pisigo

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (c *Context) Request() *http.Request {
	return c.request
}

func (c *Context) SetRequest(r *http.Request) {
	if r != nil {
		c.request = r
	}
}

func (c *Context) Method() string {
	return c.request.Method
}

func (c *Context) Path() string {
	return c.request.URL.Path
}

func (c *Context) Host() string {
	return c.request.Host
}

func (c *Context) Param(key string) string {
	return c.request.PathValue(key)
}

func (c *Context) Query(key string) string {
	return c.request.URL.Query().Get(key)
}

func (c *Context) QueryDefault(key string, fallback string) string {
	value := c.Query(key)
	if value == "" {
		return fallback
	}
	return value
}

func (c *Context) QueryInt(key string, fallback int) int {
	value := c.Query(key)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func (c *Context) HeaderGet(key string) string {
	return c.request.Header.Get(key)
}

func (c *Context) CookieGet(name string) (*http.Cookie, error) {
	return c.request.Cookie(name)
}

func (c *Context) Cookies() []*http.Cookie {
	return c.request.Cookies()
}

func (c *Context) Body() ([]byte, error) {
	return c.readBody()
}

func (c *Context) BindJSON(v any) error {
	data, err := c.readBody()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (c *Context) BindXML(v any) error {
	data, err := c.readBody()
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, v)
}

func (c *Context) FormValue(key string) string {
	if err := c.parseFormLimited(); err != nil {
		return ""
	}
	if c.request.Form == nil {
		return ""
	}
	return c.request.Form.Get(key)
}

func (c *Context) FormFile(key string) ([]byte, string, error) {
	if err := c.parseFormLimited(); err != nil {
		if isBodyTooLarge(err) {
			return nil, "", ErrRequestEntityTooLarge
		}
		return nil, "", err
	}
	file, header, err := c.request.FormFile(key)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	var reader io.Reader = file
	if c.maxBody > 0 {
		reader = io.LimitReader(file, c.maxBody+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}
	if c.maxBody > 0 && int64(len(data)) > c.maxBody {
		return nil, "", ErrRequestEntityTooLarge
	}
	return data, header.Filename, nil
}

func (c *Context) parseFormLimited() error {
	if c.request.PostForm != nil || c.request.MultipartForm != nil {
		return nil
	}
	max := c.maxBody
	if max <= 0 {
		max = 1 << 20
	}
	ct := c.HeaderGet("Content-Type")
	if strings.Contains(ct, "multipart/form-data") {
		err := c.request.ParseMultipartForm(max)
		if isBodyTooLarge(err) {
			return ErrRequestEntityTooLarge
		}
		return err
	}
	if c.request.Body != nil {
		c.request.Body = http.MaxBytesReader(c.writer, c.request.Body, max)
	}
	err := c.request.ParseForm()
	if isBodyTooLarge(err) {
		return ErrRequestEntityTooLarge
	}
	return err
}

func isBodyTooLarge(err error) bool {
	if err == nil {
		return false
	}
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return true
	}
	return strings.Contains(err.Error(), "http: request body too large")
}

func (c *Context) UserAgent() string {
	return c.request.UserAgent()
}

func (c *Context) Referer() string {
	return c.request.Referer()
}

func (c *Context) IsJSON() bool {
	return strings.Contains(c.HeaderGet("Content-Type"), "application/json")
}

func (c *Context) IsXML() bool {
	ct := c.HeaderGet("Content-Type")
	return strings.Contains(ct, "application/xml") || strings.Contains(ct, "text/xml")
}

func (c *Context) BearerToken() string {
	auth := c.HeaderGet("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return auth[7:]
	}
	return ""
}
