package service

import (
	"bytes"
	"fmt"
	"io"
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
	if req.Method == http.MethodGet {
		return nil, false
	}

	// TODO: path canonicalization necessary
	if req.URL.Path == "/graphql" && req.Method == http.MethodPost && req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			fmt.Printf("POST body: <failed to read: %v>\n", err)
		} else {
			// Restore body after reading.
			req.Body = io.NopCloser(bytes.NewReader(body))

			fmt.Printf("POST body:\n%s\n", string(body))

			// lol
			if bytes.Contains(body, []byte("mutation")) {
				return syntheticResponse(
					http.StatusForbidden,
					fmt.Sprintf("dropped due to policy violation (graphql mutation)\n"),
				), true
			} else {
				return nil, false
			}
		}
	}

	return syntheticResponse(
		http.StatusForbidden,
		fmt.Sprintf("dropped due to policy violation (%s)\n", req.Method),
	), true
}
