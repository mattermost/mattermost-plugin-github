package testutils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	// ContentTypeJSON string associated with content-type json
	ContentTypeJSON contentType = "application/json"

	// ContentTypeHTML string associated with content-type html
	ContentTypeHTML contentType = "text/html"

	// ContentTypePlain
	ContentTypePlain contentType = "text/plain; charset=utf-8"
)

// contentType type for http Content-Type
type contentType string

// Request stores http Request basic data
type Request struct {
	Method string
	URL    string
	Body   interface{}
}

// ExpectedResponse stores expected response basic data
type ExpectedResponse struct {
	StatusCode   int
	ResponseType contentType
	Body         interface{}
}

// HTTPTest encapsulates data for testing needs
type HTTPTest struct {
	*testing.T
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
	tassert := assert.New(test.T)
	var body io.Reader
	data, err := test.Encoder(request.Body)
	tassert.NoError(err)
	body = bytes.NewBuffer(data)

	req, err := http.NewRequest(request.Method, request.URL, body)
	tassert.NoError(err, "Error while creating Request")
	return req
}

// CompareHTTPResponse compares expected response with real one
func (test *HTTPTest) CompareHTTPResponse(rr *httptest.ResponseRecorder, expected ExpectedResponse) {
	tassert := assert.New(test.T)
	tassert.Equal(expected.StatusCode, rr.Code, "Http status codes are different")

	expectedBody, err := test.Encoder(expected.Body)
	tassert.NoError(err)

	tassert.Equal(string(expected.ResponseType), rr.Header().Get("Content-Type"))

	gotBody := rr.Body.Bytes()

	tassert.Equal(expectedBody, gotBody)
}
