package bunny

import (
	"encoding/json"
	"testing"
)

func TestDNSRecordUnmarshalNumericType(t *testing.T) {
	payload := []byte(`{"Id":111,"Name":"","Type":1,"Value":"127.0.0.1","Ttl":60,"Priority":0}`)

	var record DNSRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := record.Type, "1"; got != want {
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

