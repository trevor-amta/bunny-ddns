package bunny

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/trevorspencer/bunny-dynamic-dns/internal/config"
)

const (
	apiBaseURL = "https://api.bunny.net"
)

var (
	// ErrNotFound indicates the record could not be found in Bunny.
	ErrNotFound = errors.New("bunny: record not found")
)

// Client wraps the HTTP communication with Bunny's DNS API.
type Client struct {
	zoneID    string
	apiKey    string
	userAgent string
	client    *http.Client
}

// DNSRecord represents the subset of fields from Bunny we care about.
type DNSRecord struct {
	ID       int    `json:"Id"`
	Name     string `json:"Name"`
	Type     string `json:"Type"`
	Value    string `json:"Value"`
	TTL      int    `json:"Ttl"`
	Priority int    `json:"Priority"`
}

// NewClient builds a Client with sane defaults.
func NewClient(zoneID, apiKey, userAgent string) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 5 * time.Second

	return &Client{
		zoneID:    zoneID,
		apiKey:    apiKey,
		userAgent: userAgent,
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
	}
}

// GetRecord fetches the current state for a DNS record.
func (c *Client) GetRecord(ctx context.Context, recordID int) (*DNSRecord, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/dnszone/%s/records/%d", apiBaseURL, c.zoneID, recordID),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("AccessKey", c.apiKey)
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 512))
		if readErr != nil {
			return nil, fmt.Errorf("bunny api error status %d", resp.StatusCode)
		}

		return nil, fmt.Errorf("bunny api error status %d: %s", resp.StatusCode, string(body))
	}

	var record DNSRecord
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &record, nil
}

// UpdateRecord sets the record value to the provided IP address.
func (c *Client) UpdateRecord(ctx context.Context, record config.Record, ip string) error {
	body := map[string]any{
		"Type":  record.Type,
		"Name":  record.Name,
		"Value": ip,
	}

	if record.TTL > 0 {
		body["TTL"] = record.TTL
	}

	if record.Priority > 0 {
		body["Priority"] = record.Priority
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		fmt.Sprintf("%s/dnszone/%s/records/%d", apiBaseURL, c.zoneID, record.ID),
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AccessKey", c.apiKey)
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 512))
	if readErr != nil {
		return fmt.Errorf("bunny api error status %d", resp.StatusCode)
	}

	var detail map[string]any
	if err := json.Unmarshal(bodyBytes, &detail); err == nil && len(detail) > 0 {
		return fmt.Errorf("bunny api error status %d: %v", resp.StatusCode, detail)
	}

	return fmt.Errorf("bunny api error status %d: %s", resp.StatusCode, string(bodyBytes))
}
