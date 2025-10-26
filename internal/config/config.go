package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPollInterval = 120 * time.Second
	defaultUserAgent    = "bunny-ddns/0.1.2"
)

// Record describes a Bunny DNS record to be updated when the WAN IP changes.
type Record struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Priority int    `json:"priority,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
}

// Config groups all runtime settings loaded from environment variables.
type Config struct {
	PollInterval time.Duration
	APIKey       string
	ZoneID       string
	Records      []Record
	IPProviders  []string
	UserAgent    string
}

// Load builds the Config from environment variables.
func Load() (*Config, error) {
	apiKey := strings.TrimSpace(os.Getenv("BUNNY_API_KEY"))
	if apiKey == "" {
		return nil, errors.New("BUNNY_API_KEY is required")
	}

	zoneID := strings.TrimSpace(os.Getenv("BUNNY_ZONE_ID"))
	if zoneID == "" {
		return nil, errors.New("BUNNY_ZONE_ID is required")
	}

	recordsJSON := strings.TrimSpace(os.Getenv("BUNNY_RECORDS_JSON"))
	if recordsJSON == "" {
		return nil, errors.New("BUNNY_RECORDS_JSON is required")
	}

	var records []Record
	if err := json.Unmarshal([]byte(recordsJSON), &records); err != nil {
		unescaped := ""
		if candidate, unescapeErr := strconv.Unquote(recordsJSON); unescapeErr == nil {
			unescaped = candidate
		} else if candidate, wrapErr := strconv.Unquote("\"" + recordsJSON + "\""); wrapErr == nil {
			unescaped = candidate
		}

		if unescaped == "" {
			return nil, fmt.Errorf("failed to parse BUNNY_RECORDS_JSON: %w", err)
		}

		if retryErr := json.Unmarshal([]byte(unescaped), &records); retryErr != nil {
			return nil, fmt.Errorf("failed to parse BUNNY_RECORDS_JSON: %w", retryErr)
		}
	}

	if len(records) == 0 {
		return nil, errors.New("BUNNY_RECORDS_JSON must contain at least one record")
	}

	for i := range records {
		record := &records[i]

		if record.ID <= 0 {
			return nil, fmt.Errorf("record at index %d must have a positive id", i)
		}

		name := strings.TrimSpace(record.Name)
		switch {
		case name != "":
			record.Name = name
		case record.Name == "":
			// Empty name denotes the zone apex (root); leave as-is.
		default:
			return nil, fmt.Errorf("record at index %d must include name", i)
		}

		record.Type = strings.TrimSpace(record.Type)
		if record.Type == "" {
			return nil, fmt.Errorf("record at index %d must include type", i)
		}
	}

	pollInterval := defaultPollInterval
	if raw := strings.TrimSpace(os.Getenv("POLL_INTERVAL_SECONDS")); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return nil, fmt.Errorf("invalid POLL_INTERVAL_SECONDS value %q", raw)
		}

		pollInterval = time.Duration(seconds) * time.Second
	}

	providers := []string{
		"https://api.ipify.org",
		"https://ipv4.icanhazip.com",
	}

	if raw := strings.TrimSpace(os.Getenv("WAN_IP_ENDPOINTS")); raw != "" {
		custom := splitAndTrim(raw, ",")
		if len(custom) > 0 {
			providers = custom
		}
	}

	userAgent := strings.TrimSpace(os.Getenv("USER_AGENT"))
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	return &Config{
		PollInterval: pollInterval,
		APIKey:       apiKey,
		ZoneID:       zoneID,
		Records:      records,
		IPProviders:  providers,
		UserAgent:    userAgent,
	}, nil
}

func splitAndTrim(input string, sep string) []string {
	chunks := strings.Split(input, sep)
	trimmed := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if val := strings.TrimSpace(chunk); val != "" {
			trimmed = append(trimmed, val)
		}
	}
	return trimmed
}
