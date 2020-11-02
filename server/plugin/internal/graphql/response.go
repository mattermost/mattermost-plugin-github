package graphql

import (
	"fmt"
	"net/url"
	"reflect"
	"time"

	"github.com/shurcooL/githubv4"
)

// Response is the type of successful response from Github GraphQL API
type Response map[string]interface{}

const (
	TypeResponse = iota + 1
	TypeResponseSlice
	TypeOther
)

// Get returns the value corresponding to given key. If not found, error is returned.
func (r Response) Get(key string) (interface{}, error) {
	res, ok := r[key]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}

	return res, nil
}

// IsChildTypeResult checks if the type of the child is Response.
// This is useful to prevent errors before converting values with interface{} type to Response.
func (r Response) IsChildTypeResult(key string) bool {
	res, err := r.Get(key)

	if err != nil {
		return false
	}

	return reflect.TypeOf(res) == reflect.TypeOf(Response{})
}

// GetChildType checks if the type of the child is Response, []Response or anything else.
func (r Response) GetChildType(key string) int {
	res, err := r.Get(key)
	if err != nil {
		return TypeOther
	}

	switch reflect.TypeOf(res) {
	case reflect.TypeOf(Response{}):
		return TypeResponse
	case reflect.TypeOf([]Response{}):
		return TypeResponseSlice
	default:
		return TypeOther
	}
}

// convertToResponse loops through the fields of response struct and converts it to Response
func convertToResponse(v interface{}) (Response, error) {
	if v == nil {
		return nil, fmt.Errorf("nil value given")
	}
	result := make(Response)
	var s reflect.Value
	var t reflect.Type

	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		s = reflect.Indirect(reflect.ValueOf(v))
		t = reflect.TypeOf(v).Elem()
	} else {
		s = reflect.ValueOf(v)
		t = reflect.TypeOf(v)
	}

	for i := 0; i < s.NumField(); i++ {
		key := t.Field(i).Name
		val := s.Field(i)

		switch val.Kind() {
		case reflect.Struct:
			if val.Type() == reflect.TypeOf(time.Time{}) {
				result[key] = val.Interface().(time.Time).Format("2006-01-02, 15:04:05")
			} else {
				res, err := convertToResponse(val.Interface())
				if err != nil {
					return nil, err
				}
				result[key] = res
			}
		case reflect.Slice:
			var nestedChildren []Response
			child := reflect.ValueOf(val.Interface())

			for j := 0; j < child.Len(); j++ {
				f, err := convertToResponse(child.Index(j).Interface())
				if err != nil {
					return nil, err
				}
				nestedChildren = append(nestedChildren, f)
			}
			result[key] = nestedChildren
		case reflect.TypeOf((*githubv4.URI)(nil)).Kind():
			result[key] = val.Interface().(*url.URL).String()
		default:
			result[key] = val.Interface()
		}
	}

	return result, nil
}
