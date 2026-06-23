package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// RequestModifier is called after the proxy receives a client request and
// before it is forwarded to the origin server. Return an error to abort the
// request with HTTP 502 Bad Gateway.
type RequestModifier func(req *http.Request) error

// ResponseModifier is called after the origin server responds and before the
// response is sent back to the client. Return an error to abort with HTTP 502.
type ResponseModifier func(resp *http.Response) error

// RequestInterceptor may short-circuit a request before it reaches the origin.
// Return a non-nil response and true to respond locally without forwarding.
type RequestInterceptor func(req *http.Request) (*http.Response, bool)

// Proxy is an HTTP forward proxy that forwards client traffic to origin servers.
type Proxy struct {
	ModifyRequest    RequestModifier
	ModifyResponse   ResponseModifier
	InterceptRequest RequestInterceptor

	ca     *CA
	client *http.Client
}

// New returns a Proxy that intercepts TLS using the given CA.
func New(ca *CA) *Proxy {
	return &Proxy{
		ca: ca,
		client: &http.Client{
			// Do not follow redirects; the client decides how to handle them.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				Proxy:              nil, // never chain through another proxy
				ForceAttemptHTTP2:  false,
				TLSNextProto:       make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			},
		},
	}
}

// ServeHTTP implements http.Handler.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
		return
	}
	p.handleHTTP(w, r)
}

func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	outReq, err := p.buildOutboundRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := p.roundTrip(outReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	removeHopByHopHeaders(w.Header())
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *Proxy) roundTrip(outReq *http.Request) (*http.Response, error) {
	if p.InterceptRequest != nil {
		if resp, ok := p.InterceptRequest(outReq); ok {
			return resp, nil
		}
	}

	if p.ModifyRequest != nil {
		if err := p.ModifyRequest(outReq); err != nil {
			return nil, err
		}
	}

	resp, err := p.client.Do(outReq)
	if err != nil {
		return nil, fmt.Errorf("failed to reach origin: %w", err)
	}

	if p.ModifyResponse != nil {
		if err := p.ModifyResponse(resp); err != nil {
			resp.Body.Close()
			return nil, err
		}
	}

	return resp, nil
}

func (p *Proxy) buildOutboundRequest(r *http.Request) (*http.Request, error) {
	targetURL, err := requestTargetURL(r)
	if err != nil {
		return nil, err
	}

	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		return nil, fmt.Errorf("create outbound request: %w", err)
	}

	copyHeader(outReq.Header, r.Header)
	removeHopByHopHeaders(outReq.Header)
	outReq.Header.Del("Proxy-Connection")

	outReq.ContentLength = r.ContentLength
	outReq.Host = outReq.URL.Host

	return outReq, nil
}

func requestTargetURL(r *http.Request) (string, error) {
	if r.URL != nil && r.URL.IsAbs() {
		return r.URL.String(), nil
	}

	// Some clients send an absolute URI in RequestURI instead of URL.
	if strings.HasPrefix(r.RequestURI, "http://") || strings.HasPrefix(r.RequestURI, "https://") {
		return r.RequestURI, nil
	}

	if r.Host == "" {
		return "", fmt.Errorf("missing target host")
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	u := url.URL{
		Scheme:   scheme,
		Host:     r.Host,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
	}
	return u.String(), nil
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

var hopByHopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func removeHopByHopHeaders(header http.Header) {
	for _, key := range hopByHopHeaders {
		header.Del(key)
	}

	if connectionTokens := header.Get("Connection"); connectionTokens != "" {
		for _, token := range strings.Split(connectionTokens, ",") {
			header.Del(strings.TrimSpace(token))
		}
	}
}
