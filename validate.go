package pisigo

import (
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validateOnce sync.Once
	validate     *validator.Validate
)

func Validator() *validator.Validate {
	validateOnce.Do(func() {
		validate = validator.New()
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" || name == "" {
				return fld.Name
			}
			return name
		})
	})
	return validate
}

func Validate(v any) error {
	err := Validator().Struct(v)
	if err == nil {
		return nil
	}
	details := map[string]any{}
	if errs, ok := err.(validator.ValidationErrors); ok {
		fields := map[string]string{}
		for _, fe := range errs {
			fields[fe.Field()] = fe.Tag()
		}
		details["fields"] = fields
	}
	return NewHTTPError(400, "validation failed").WithDetails(details)
}

func (c *Context) Bind(v any) error {
	ct := c.HeaderGet("Content-Type")
	switch {
	case strings.Contains(ct, "application/json"):
		if err := c.BindJSON(v); err != nil {
			return NewHTTPError(400, "invalid json")
		}
	case strings.Contains(ct, "xml"):
		if err := c.BindXML(v); err != nil {
			return NewHTTPError(400, "invalid xml")
		}
	default:
		if err := c.BindJSON(v); err != nil {
			return NewHTTPError(400, "invalid body")
		}
	}
	return Validate(v)
}

func (c *Context) Validate(v any) error {
	return Validate(v)
}
