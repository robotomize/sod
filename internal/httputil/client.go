package httputil

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// NewClient returns a http.Client using the specified http.RoundTripper.
func newClient(rt http.RoundTripper) *http.Client {
	return &http.Client{Transport: rt}
}

func NewClientFromConfig(cfg HTTPClientConfig, disableKeepAlives bool) (*http.Client, error) {
	rt, err := NewRoundTripperFromConfig(cfg, disableKeepAlives)
	if err != nil {
		return nil, err
	}
	return newClient(rt), nil
}

// NewRoundTripperFromConfig returns a new HTTP RoundTripper configured for the
// given config.HTTPClientConfig
func NewRoundTripperFromConfig(cfg HTTPClientConfig, disableKeepAlives bool) (http.RoundTripper, error) {
	var rt http.RoundTripper = &http.Transport{
		MaxIdleConns:          20000,
		MaxIdleConnsPerHost:   1000,
		DisableKeepAlives:     disableKeepAlives,
		DisableCompression:    true,
		IdleConnTimeout:       5 * time.Minute,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	// If a bearer token is provided, create a round tripper that will set the
	// Authorization header correctly on each request.
	if len(cfg.BearerToken) > 0 {
		rt = NewBearerAuthRoundTripper(cfg.BearerToken, rt)
	}

	if cfg.BasicAuth != nil {
		rt = NewBasicAuthRoundTripper(cfg.BasicAuth.Username, cfg.BasicAuth.Password, rt)
	}

	return rt, nil
}

type bearerAuthRoundTripper struct {
	bearerToken string
	rt          http.RoundTripper
}

// NewBearerAuthRoundTripper adds the provided bearer token to a request unless the authorization
// header has already been set.
func NewBearerAuthRoundTripper(token string, rt http.RoundTripper) http.RoundTripper {
	return &bearerAuthRoundTripper{token, rt}
}

func (rt *bearerAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.Header.Get("Authorization")) == 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rt.bearerToken))
	}
	return rt.rt.RoundTrip(req)
}

type basicAuthRoundTripper struct {
	username string
	password string
	rt       http.RoundTripper
}

// NewBasicAuthRoundTripper will apply a BASIC auth authorization header to a request unless it has
// already been set.
func NewBasicAuthRoundTripper(username string, password string, rt http.RoundTripper) http.RoundTripper {
	return &basicAuthRoundTripper{username, password, rt}
}

func (rt *basicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.Header.Get("Authorization")) != 0 {
		return rt.rt.RoundTrip(req)
	}
	req.SetBasicAuth(rt.username, strings.TrimSpace(rt.password))
	return rt.rt.RoundTrip(req)
}
