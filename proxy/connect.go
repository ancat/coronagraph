package proxy

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func (p *Proxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, bufrw, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "failed to hijack connection: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	fmt.Fprintf(bufrw, "HTTP/1.1 200 Connection Established\r\n\r\n")
	if err := bufrw.Flush(); err != nil {
		return
	}

	conn := &bufferedConn{Conn: clientConn, r: bufrw.Reader}

	tlsConfig := &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			host := hello.ServerName
			if host == "" {
				host = r.Host
			}
			return p.ca.CertificateForHost(host)
		},
		NextProtos: []string{"http/1.1"},
	}

	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return
	}
	defer tlsConn.Close()

	br := bufio.NewReader(tlsConn)
	for {
		req, err := http.ReadRequest(br)
		if err != nil {
			return
		}

		if err := p.handleMITMRequest(tlsConn, req); err != nil {
			return
		}

		if req.Close || strings.EqualFold(req.Header.Get("Connection"), "close") {
			return
		}
	}
}

func (p *Proxy) handleMITMRequest(w io.Writer, r *http.Request) error {
	defer r.Body.Close()

	outReq, err := p.buildMITMOutboundRequest(r)
	if err != nil {
		writeWireError(w, http.StatusBadRequest, err.Error())
		return err
	}

	resp, err := p.roundTrip(outReq)
	if err != nil {
		writeWireError(w, http.StatusBadGateway, err.Error())
		return err
	}
	defer resp.Body.Close()

	return writeMITMResponse(w, resp)
}

func writeMITMResponse(w io.Writer, resp *http.Response) error {
	removeHopByHopHeaders(resp.Header)
	resp.Header.Del("Transfer-Encoding")

	// ContentLength is set by net/http when the origin sends a plain
	// Content-Length (or after any auto-decompression, -1 if unknown).
	if resp.ContentLength >= 0 {
		resp.Header.Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
		if err := writeResponseHeaders(w, resp.StatusCode, resp.Header); err != nil {
			return err
		}
		_, err := io.Copy(w, resp.Body)
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	if err := writeResponseHeaders(w, resp.StatusCode, resp.Header); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

func writeResponseHeaders(w io.Writer, statusCode int, header http.Header) error {
	if _, err := fmt.Fprintf(w, "HTTP/1.1 %d %s\r\n", statusCode, statusText(statusCode)); err != nil {
		return err
	}
	if err := header.Write(w); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\r\n")
	return err
}

func statusText(code int) string {
	if text := http.StatusText(code); text != "" {
		return text
	}
	return fmt.Sprintf("status code %d", code)
}

func (p *Proxy) buildMITMOutboundRequest(r *http.Request) (*http.Request, error) {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	target := "https://" + host + r.URL.RequestURI()
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	if err != nil {
		return nil, fmt.Errorf("create outbound request: %w", err)
	}

	copyHeader(outReq.Header, r.Header)
	removeHopByHopHeaders(outReq.Header)
	outReq.Header.Del("Proxy-Connection")

	outReq.ContentLength = r.ContentLength
	outReq.Host = host

	return outReq, nil
}

func writeWireError(w io.Writer, code int, msg string) {
	resp := &http.Response{
		StatusCode:    code,
		ContentLength: int64(len(msg)),
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(msg)),
	}
	resp.Header.Set("Content-Type", "text/plain; charset=utf-8")
	writeMITMResponse(w, resp)
}

type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	return c.r.Read(p)
}
