package service

import (
	"io"
	"net"
	"net/http"
	"strings"
)

func requestHost(req *http.Request) string {
	host := req.URL.Host
	if host == "" {
		host = req.Host
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

func syntheticResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode:    statusCode,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}
