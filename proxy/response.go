package proxy

import (
	"io"
	"net/http"
	"strings"
)

// SyntheticResponse builds a local response without contacting an origin server.
func SyntheticResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode:    statusCode,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}
