package integration

import (
	"bytes"
	"context"
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

func (c *Client) Collect(r Request) (*http.Response, error) {
	b, err := json.Marshal(&r)
	if err != nil {
		return nil, fmt.Errorf("unable marshal collect request: %w", err)
	}
	reader := bytes.NewReader(b)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/collect", reader)
	if err != nil {
		return nil, fmt.Errorf("create new request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error with sending request: %w", err)
	}
	defer resp.Body.Close()
	return resp, nil
}

func (c *Client) Predict(r Request) (*http.Response, error) {
	b, err := json.Marshal(&r)
	if err != nil {
		return nil, fmt.Errorf("unable marshal collect request: %w", err)
	}

	reader := bytes.NewReader(b)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/predict", reader)
	if err != nil {
		return nil, fmt.Errorf("create new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error with sending request: %w", err)
	}

	defer resp.Body.Close()
	return resp, nil
}

func (c *Client) Health() (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	if err != nil {
		return nil, fmt.Errorf("create new request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	return resp, nil
}
