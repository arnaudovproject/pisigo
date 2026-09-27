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
	locked := c.rlock()
	defer c.runlock(locked)
	return c.writer
}

func (c *Context) SetWriter(w http.ResponseWriter) {
	if w == nil {
		return
	}
	locked := c.lock()
	defer c.unlock(locked)
	c.writer = w
}

func (c *Context) Header(key string, value string) {
	c.Response().Header().Set(key, value)
}

func (c *Context) Status(status int) {
	locked := c.lock()
	c.status = status
	w := c.writer
	already := c.written
	if !already {
		c.written = true
	}
	c.unlock(locked)
	if !already && w != nil {
		w.WriteHeader(status)
	}
}

func (c *Context) writeStatus(status int) {
	locked := c.lock()
	if timedOutWriter(c.writer) {
		c.unlock(locked)
		return
	}
	w := c.writer
	already := c.written
	c.status = status
	if !already {
		c.written = true
	}
	c.unlock(locked)
	if !already && w != nil {
		w.WriteHeader(status)
	}
}

func (c *Context) String(status int, value string) error {
	if timedOutWriter(c.Response()) {
		return nil
	}
	c.Response().Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.writeStatus(status)
	_, err := fmt.Fprint(c.Response(), value)
	return err
}

func (c *Context) HTML(status int, value string) error {
	if timedOutWriter(c.Response()) {
		return nil
	}
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.writeStatus(status)
	_, err := fmt.Fprint(c.Response(), value)
	return err
}

func (c *Context) JSON(status int, data any) error {
	if timedOutWriter(c.Response()) {
		return nil
	}
	c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
	c.writeStatus(status)
	return json.NewEncoder(c.Response()).Encode(data)
}

func (c *Context) XML(status int, data any) error {
	if timedOutWriter(c.Response()) {
		return nil
	}
	c.Response().Header().Set("Content-Type", "application/xml; charset=utf-8")
	c.writeStatus(status)
	return xml.NewEncoder(c.Response()).Encode(data)
}

func (c *Context) Data(status int, contentType string, data []byte) error {
	if timedOutWriter(c.Response()) {
		return nil
	}
	c.Response().Header().Set("Content-Type", contentType)
	c.writeStatus(status)
	_, err := c.Response().Write(data)
	return err
}

func (c *Context) NoContent() error {
	c.writeStatus(http.StatusNoContent)
	return nil
}

func (c *Context) Redirect(status int, location string) error {
	http.Redirect(c.writer, c.request, location, status)
	c.syncFromWriter()
	if !c.written {
		c.status = status
		c.written = true
	}
	return nil
}

func (c *Context) File(path string) error {
	http.ServeFile(c.writer, c.request, path)
	c.syncFromWriter()
	return nil
}

func (c *Context) Download(filePath string, filename string) error {
	safe := sanitizeContentDispositionFilename(filename)
	c.writer.Header().Set(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, safe),
	)
	http.ServeFile(c.writer, c.request, filePath)
	c.syncFromWriter()
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

func timedOutWriter(w http.ResponseWriter) bool {
	type timedOutFlag interface {
		TimedOut() bool
	}
	for w != nil {
		if t, ok := w.(timedOutFlag); ok && t.TimedOut() {
			return true
		}
		u, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return false
		}
		w = u.Unwrap()
	}
	return false
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
