package ip

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Provider queries a list of HTTP endpoints to discover the current WAN IP.
type Provider struct {
	endpoints []string
	client    *http.Client
	userAgent string
}

// NewProvider builds a Provider with reasonable defaults.
func NewProvider(endpoints []string, userAgent string) *Provider {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 5 * time.Second

	return &Provider{
		endpoints: endpoints,
		client: &http.Client{
			Timeout:   7 * time.Second,
			Transport: transport,
		},
		userAgent: userAgent,
	}
}

// CurrentIP returns the first successful response from the configured endpoints.
func (p *Provider) CurrentIP(ctx context.Context) (string, error) {
	var errs []string

	for _, endpoint := range p.endpoints {
		ip, err := p.fetchOnce(ctx, endpoint)
		if err == nil {
			return ip, nil
		}

		errs = append(errs, fmt.Sprintf("%s: %v", endpoint, err))
	}

	return "", fmt.Errorf("all IP providers failed: %s", strings.Join(errs, "; "))
}

func (p *Provider) fetchOnce(ctx context.Context, endpoint string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("request build failed: %w", err)
	}

	if p.userAgent != "" {
		req.Header.Set("User-Agent", p.userAgent)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}

	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP payload %q", ip)
	}

	return ip, nil
}
