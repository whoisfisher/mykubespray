package httpx

import (
	"crypto/tls"
	"net/http"
	"strings"
)

type CustomTransport struct {
	http.RoundTripper
}

func (t *CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.Scheme, "https") {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		return transport.RoundTrip(req)
	}

	// For HTTP, use the default transport
	return http.DefaultTransport.RoundTrip(req)
}
