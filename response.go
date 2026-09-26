// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pisigo

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

func (c *Context) Response() http.ResponseWriter {
	return c.writer
}

func (c *Context) SetWriter(w http.ResponseWriter) {
	if w != nil {
		c.writer = w
	}
}

func (c *Context) Header(key string, value string) {
	c.writer.Header().Set(key, value)
}

func (c *Context) Status(status int) {
	c.status = status
	if !c.written {
		c.writer.WriteHeader(status)
		c.written = true
	}
}

func (c *Context) writeStatus(status int) {
	c.status = status
	if !c.written {
		c.writer.WriteHeader(status)
		c.written = true
	}
}

func (c *Context) String(status int, value string) error {
	c.writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.writeStatus(status)
	_, err := fmt.Fprint(c.writer, value)
	return err
}

func (c *Context) HTML(status int, value string) error {
	c.writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.writeStatus(status)
	_, err := fmt.Fprint(c.writer, value)
	return err
}

func (c *Context) JSON(status int, data any) error {
	c.writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.writeStatus(status)
	return json.NewEncoder(c.writer).Encode(data)
}

func (c *Context) XML(status int, data any) error {
	c.writer.Header().Set("Content-Type", "application/xml; charset=utf-8")
	c.writeStatus(status)
	return xml.NewEncoder(c.writer).Encode(data)
}

func (c *Context) Data(status int, contentType string, data []byte) error {
	c.writer.Header().Set("Content-Type", contentType)
	c.writeStatus(status)
	_, err := c.writer.Write(data)
	c.written = true
	return err
}

func (c *Context) NoContent() error {
	c.writeStatus(http.StatusNoContent)
	return nil
}

func (c *Context) Redirect(status int, location string) error {
	http.Redirect(c.writer, c.request, location, status)
	c.status = status
	c.written = true
	return nil
}

func (c *Context) File(path string) error {
	http.ServeFile(c.writer, c.request, path)
	c.written = true
	return nil
}

func (c *Context) Download(filePath string, filename string) error {
	safe := sanitizeContentDispositionFilename(filename)
	c.writer.Header().Set(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, safe),
	)
	http.ServeFile(c.writer, c.request, filePath)
	c.written = true
	return nil
}

func sanitizeContentDispositionFilename(name string) string {
	name = path.Base(strings.ReplaceAll(strings.ReplaceAll(name, "\\", "/"), "\x00", ""))
	name = strings.Map(func(r rune) rune {
		switch r {
		case '"', '\r', '\n', '\t':
			return -1
		default:
			return r
		}
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "download"
	}
	return name
}

func (c *Context) Cookie(cookie *http.Cookie) {
	http.SetCookie(c.writer, cookie)
}

func (c *Context) SetCookie(
	name string,
	value string,
	maxAge int,
	path string,
	secure bool,
	httpOnly bool,
) {
	http.SetCookie(c.writer, &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     path,
		Secure:   secure,
		HttpOnly: httpOnly,
		SameSite: http.SameSiteLaxMode,
	})
}

func (c *Context) DeleteCookie(name string, path string) {
	http.SetCookie(c.writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func (c *Context) Error(status int, message string) error {
	return NewHTTPError(status, message)
}

func (c *Context) Stream(status int, contentType string, reader io.Reader) error {
	c.writer.Header().Set("Content-Type", contentType)
	c.writeStatus(status)
	_, err := io.Copy(c.writer, reader)
	return err
}
