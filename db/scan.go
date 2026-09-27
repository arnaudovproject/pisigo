package db

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

func scanOne(rows *Rows, dest any) error {
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return ErrNoRows
	}
	if err := scanDestination(rows, dest); err != nil {
		return err
	}
	if rows.Next() {
		return errors.New("db: expected one row, got more")
	}
	return rows.Err()
}

func scanAll(rows *Rows, dest any) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Slice {
		return errors.New("db: Select dest must be pointer to slice")
	}
	slice := v.Elem()
	elemType := slice.Type().Elem()
	for rows.Next() {
		var elem reflect.Value
		if elemType.Kind() == reflect.Ptr {
			elem = reflect.New(elemType.Elem())
			if err := scanDestination(rows, elem.Interface()); err != nil {
				return err
			}
		} else {
			elem = reflect.New(elemType)
			if err := scanDestination(rows, elem.Interface()); err != nil {
				return err
			}
			elem = elem.Elem()
		}
		slice.Set(reflect.Append(slice, elem))
	}
	return rows.Err()
}

func scanDestination(rows *Rows, dest any) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.New("db: dest must be non-nil pointer")
	}
	v = v.Elem()

	switch d := dest.(type) {
	case *map[string]any:
		m, err := scanMap(rows)
		if err != nil {
			return err
		}
		*d = m
		return nil
	}

	if v.Kind() == reflect.Struct {
		return scanStruct(rows, v)
	}

	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(cols) != 1 {
		return fmt.Errorf("db: expected 1 column for scalar scan, got %d", len(cols))
	}
	return rows.Scan(dest)
}

func scanMap(rows *Rows) (map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range values {
		ptrs[i] = &values[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	out := make(map[string]any, len(cols))
	for i, col := range cols {
		out[col] = normalize(values[i])
	}
	return out, nil
}

func scanStruct(rows *Rows, v reflect.Value) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	fields := structFields(v.Type())
	values := make([]any, len(cols))
	for i, col := range cols {
		if idx, ok := fields[strings.ToLower(col)]; ok {
			values[i] = v.Field(idx).Addr().Interface()
		} else {
			var skip any
			values[i] = &skip
		}
	}
	return rows.Scan(values...)
}

var fieldCache sync.Map

func structFields(t reflect.Type) map[string]int {
	if cached, ok := fieldCache.Load(t); ok {
		return cached.(map[string]int)
	}
	out := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		tag := f.Tag.Get("db")
		if tag == "-" {
			continue
		}
		name := tag
		if name == "" {
			name = f.Name
		} else if i := strings.IndexByte(name, ','); i >= 0 {
			name = name[:i]
		}
		out[strings.ToLower(name)] = i
	}
	fieldCache.Store(t, out)
	return out
}

func normalize(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case []byte:
		return string(x)
	case time.Time:
		return x
	default:
		return x
	}
}

func IsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrNoRows)
}
