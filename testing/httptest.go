// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pisigotest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/arnaudovproject/pisigo"
)

type Response struct {
	Code    int
	Header  http.Header
	Body    []byte
	Raw     *httptest.ResponseRecorder
}

func (r *Response) String() string {
	return string(r.Body)
}

func (r *Response) JSON(dest any) error {
	return json.Unmarshal(r.Body, dest)
}

type Request struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    any
}

func Do(app *pisigo.App, req Request) *Response {
	var bodyReader io.Reader
	switch b := req.Body.(type) {
	case nil:
	case string:
		bodyReader = strings.NewReader(b)
	case []byte:
		bodyReader = bytes.NewReader(b)
	default:
		data, _ := json.Marshal(b)
		bodyReader = bytes.NewReader(data)
		if req.Headers == nil {
			req.Headers = map[string]string{}
		}
		if _, ok := req.Headers["Content-Type"]; !ok {
			req.Headers["Content-Type"] = "application/json"
		}
	}
	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httpReq)
	return &Response{
		Code:   rec.Code,
		Header: rec.Header(),
		Body:   rec.Body.Bytes(),
		Raw:    rec,
	}
}

func GET(app *pisigo.App, path string) *Response {
	return Do(app, Request{Method: http.MethodGet, Path: path})
}

func POST(app *pisigo.App, path string, body any) *Response {
	return Do(app, Request{Method: http.MethodPost, Path: path, Body: body})
}
