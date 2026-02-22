package util

import (
	"net/http"
	"net/url"
)

// NewProxyFunc creates a proxy function based on configuration.
// If no proxy URLs are provided, falls back to environment variables.
func NewProxyFunc(httpProxy, httpsProxy, noProxy string) func(*http.Request) (*url.URL, error) {
	if httpProxy == "" && httpsProxy == "" {
		return http.ProxyFromEnvironment
	}

	return func(req *http.Request) (*url.URL, error) {
		if req.URL.Scheme == "https" && httpsProxy != "" {
			return url.Parse(httpsProxy)
		}
		if httpProxy != "" {
			return url.Parse(httpProxy)
		}
		return http.ProxyFromEnvironment(req)
	}
}
