package util

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
)

const (
	// ContentTypeJSON string associated with content-type json
	ContentTypeJSON ContentType = "application/json"

	// ContentTypeHTML string associated with content-type html
	ContentTypeHTML ContentType = "text/html"

	// ContentTypePlain
	ContentTypePlain ContentType = "text/plain; charset=utf-8"
)

// ContentType type for http Content-Type
type ContentType string

// Request stores http Request basic data
type Request struct {
	Method string
	URL    string
	Body   interface{}
}

// ExpectedResponse stores expected responce basic data
type ExpectedResponse struct {
	StatusCode   int
	ResponseType ContentType
	Body         interface{}
}

// HTTPTest encapsulates data for testing needs
type HTTPTest struct {
	*assert.Assertions
	Encoder func(interface{}) ([]byte, error)
}

// EncodeJSON encodes json data in bytes
func EncodeJSON(data interface{}) ([]byte, error) {
	if data == nil {
		return []byte{}, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return []byte{}, errors.Wrap(err, "Error while encoding json")
	}

	return b, nil

}

// EncodeJSON encodes json data in bytes
func EncodeString(data interface{}) ([]byte, error) {
	if data == nil {
		return []byte{}, nil
	}

	if s, ok := data.(string); ok {
		return []byte(s), nil
	}

	return []byte{}, errors.New("error while decoding string")
}

// CreateHTTPRequest creates http Request with basic data
func (test *HTTPTest) CreateHTTPRequest(request Request) *http.Request {
	var body io.Reader
	data, err := test.Encoder(request.Body)
	test.NoError(err)
	body = bytes.NewBuffer(data)

	req, err := http.NewRequest(request.Method, request.URL, body)
	test.NoError(err, "Error while creating Request")
	return req
}

// CompareHTTPResponse compares expected response with real one
func (test *HTTPTest) CompareHTTPResponse(rr *httptest.ResponseRecorder, expected ExpectedResponse) {

	test.Equal(expected.StatusCode, rr.Code, "Http status codes are different")

	expectedBody, err := test.Encoder(expected.Body)
	test.NoError(err)

	test.Equal(string(expected.ResponseType), rr.Header().Get("Content-Type"))

	gotBody := rr.Body.Bytes()

	test.Equal(expectedBody, gotBody)
}
