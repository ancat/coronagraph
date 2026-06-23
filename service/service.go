package service

import "net/http"

// Service represents a web API whose traffic the proxy may authenticate or block.
type Service interface {
	Matches(req *http.Request) bool
	InsertAuthentication(req *http.Request)
	DropRequest(req *http.Request) (*http.Response, bool)
	Name() string
}
