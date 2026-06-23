package service

import (
	"fmt"
	"net/http"
)

var githubHosts = map[string]bool{
	"github.com":     true,
	"api.github.com": true,
}

// GitHub handles proxy behavior for GitHub hosts.
type GitHub struct {
	token string
}

// NewGitHub returns a GitHub service configured with the given personal access token.
func NewGitHub(token string) *GitHub {
	return &GitHub{token: token}
}

func (s *GitHub) Name() string {
	return "github"
}

// Matches reports whether req targets a GitHub host.
func (s *GitHub) Matches(req *http.Request) bool {
	return githubHosts[requestHost(req)]
}

// InsertAuthentication adds a GitHub personal access token as an Authorization header.
func (s *GitHub) InsertAuthentication(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("token %s", s.token))
}

// DropRequest blocks state-changing requests. GET is allowed without confirmation.
func (s *GitHub) DropRequest(req *http.Request) (*http.Response, bool) {
	if req.Method != http.MethodGet {
		return syntheticResponse(http.StatusForbidden, fmt.Sprintf("dropped due to policy violation (%s)\n", req.Method)), true
	}
	return nil, false
}
