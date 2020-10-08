package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type prefixRoundTripper struct {
	addr string
	rt   http.RoundTripper
}

func (p *prefixRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	u := r.URL
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	if u.Host == "" {
		u.Host = p.addr
	}

	return p.rt.RoundTrip(r)
}

func NewClient(addr string) *Client {
	return &Client{client: &http.Client{Transport: &prefixRoundTripper{addr: addr, rt: http.DefaultTransport}}}
}

type Client struct {
	client *http.Client
}

func (c *Client) Collect(req Request) (*http.Response, error) {
	b, err := json.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("unable marshal collect request: %w", err)
	}
	reader := bytes.NewReader(b)
	resp, err := c.client.Post("/collect", "application/json", reader)
	if err != nil {
		return nil, fmt.Errorf("error with sending request: %w", err)
	}
	defer resp.Body.Close()
	return resp, nil
}

func (c *Client) Predict(req Request) (*http.Response, error) {
	b, err := json.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("unable marshal collect request: %w", err)
	}
	reader := bytes.NewReader(b)
	resp, err := c.client.Post("/predict", "application/json", reader)
	if err != nil {
		return nil, fmt.Errorf("error with sending request: %w", err)
	}
	defer resp.Body.Close()
	return resp, nil
}

func (c *Client) Health() (*http.Response, error) {
	resp, err := c.client.Get("/health")
	if err != nil {
		return nil, fmt.Errorf("error with sending request: %w", err)
	}
	return resp, nil
}
