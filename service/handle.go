package service

import (
	"fmt"
	"net/http"
)

// Authenticator prompts the user to approve a sensitive request.
type Authenticator func(reason string) bool

// Process applies service rules to an outbound request.
// State-changing requests require confirmation; on success credentials are
// inserted and the request proceeds. On failure the drop response is returned.
// Non-state-changing matched requests are authenticated and allowed through.
func Process(services []Service, req *http.Request, confirm Authenticator) (resp *http.Response, stop bool, matched Service) {
	for _, svc := range services {
		if !svc.Matches(req) {
			continue
		}
		if resp, drop := svc.DropRequest(req); drop {
			reason := fmt.Sprintf("Allow %s %s on %s?", req.Method, req.URL.Path, svc.Name())
			if confirm != nil && confirm(reason) {
				svc.InsertAuthentication(req)
				return nil, false, svc
			}
			return resp, true, svc
		}
		svc.InsertAuthentication(req)
		return nil, false, svc
	}
	return nil, false, nil
}
