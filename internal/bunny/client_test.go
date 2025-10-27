package bunny

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/trevorspencer/bunny-dynamic-dns/internal/config"
)

func TestDNSRecordUnmarshalNumericType(t *testing.T) {
	payload := []byte(`{"Id":111,"Name":"","Type":1,"Value":"127.0.0.1","Ttl":60,"Priority":0}`)

	var record DNSRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := record.Type, "AAAA"; got != want {
		t.Fatalf("Type mismatch: got %q want %q", got, want)
	}
}

func TestDNSRecordUnmarshalStringType(t *testing.T) {
	payload := []byte(`{"Id":111,"Name":"","Type":"A","Value":"127.0.0.1","Ttl":60,"Priority":0}`)

	var record DNSRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := record.Type, "A"; got != want {
		t.Fatalf("Type mismatch: got %q want %q", got, want)
	}
}

func TestDNSRecordUnmarshalNumericTypeFallback(t *testing.T) {
	payload := []byte(`{"Id":111,"Name":"","Type":99,"Value":"127.0.0.1","Ttl":60,"Priority":0}`)

	var record DNSRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := record.Type, "99"; got != want {
		t.Fatalf("Type mismatch: got %q want %q", got, want)
	}
}

func TestDNSRecordUnmarshalStringNormalization(t *testing.T) {
	payload := []byte(`{"Id":111,"Name":"","Type":"a","Value":"127.0.0.1","Ttl":60,"Priority":0}`)

	var record DNSRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := record.Type, "A"; got != want {
		t.Fatalf("Type mismatch: got %q want %q", got, want)
	}
}

func TestUpdateRecordFallbackOn405(t *testing.T) {
	t.Helper()

	client := NewClient("zone-id", "key", "agent")

	var callCount int
	client.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			callCount++

			switch callCount {
			case 1:
				if req.Method != http.MethodPut {
					t.Fatalf("expected first request to be PUT, got %s", req.Method)
				}
				body := `{"Message":"The requested resource does not support http method 'PUT'."}`
				resp := &http.Response{
					StatusCode: http.StatusMethodNotAllowed,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(body)),
					Request:    req,
				}
				resp.Header.Set("Allow", "GET, POST")
				return resp, nil
			case 2:
				if req.Method != http.MethodPost {
					t.Fatalf("expected fallback request to be POST, got %s", req.Method)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("{}")),
					Request:    req,
				}, nil
			default:
				t.Fatalf("unexpected extra request #%d", callCount)
				return nil, nil
			}
		}),
	}

	err := client.UpdateRecord(context.Background(), config.Record{
		ID:   123,
		Type: "A",
		Name: "",
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Fatalf("expected 2 HTTP calls, got %d", callCount)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
