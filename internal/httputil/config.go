package httputil

import "fmt"

type HTTPClientConfig struct {
	BasicAuth   *BasicAuth `json:"basicAuth,omitempty"`
	BearerToken string     `yaml:"bearerToken,omitempty"`
}

func (c *HTTPClientConfig) Validate() error {
	if len(c.BearerToken) > 0 {
		return fmt.Errorf("at most one of bearer_token & bearer_token_file must be configured")
	}
	if c.BasicAuth != nil && len(c.BearerToken) > 0 {
		return fmt.Errorf("at most one of basic_auth, bearer_token & bearer_token_file must be configured")
	}
	if c.BasicAuth != nil && c.BasicAuth.Password != "" {
		return fmt.Errorf("at most one of basic_auth password & password_file must be configured")
	}
	return nil
}

type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password,omitempty"`
}
