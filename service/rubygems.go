package service

import "net/http"

const rubyGemsHost = "rubygems.org"

// RubyGems handles proxy behavior for rubygems.org.
type RubyGems struct {
	apiKey string
}

// NewRubyGems returns a RubyGems service configured with the given API key.
func NewRubyGems(apiKey string) *RubyGems {
	return &RubyGems{apiKey: apiKey}
}

func (s *RubyGems) Name() string {
	return "rubygems"
}

// Matches reports whether req targets rubygems.org.
func (s *RubyGems) Matches(req *http.Request) bool {
	return requestHost(req) == rubyGemsHost
}

// InsertAuthentication adds the RubyGems API key as an Authorization header.
func (s *RubyGems) InsertAuthentication(req *http.Request) {
	req.Header.Set("Authorization", s.apiKey)
}

// DropRequest blocks state-changing requests. POST is treated as state-changing.
func (s *RubyGems) DropRequest(req *http.Request) (*http.Response, bool) {
	if req.Method != http.MethodGet {
		return syntheticResponse(http.StatusForbidden, "rubygems dropped due to policy violation\n"), true
	}
	return nil, false
}
