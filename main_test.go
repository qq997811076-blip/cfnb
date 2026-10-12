package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestParseAdaptiveTextNodes(t *testing.T) {
	got := parseAdaptive("192.0.2.1:443#US\n198.51.100.2:8443#JP")
	want := []string{"192.0.2.1:443#US", "198.51.100.2:8443#JP"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseAdaptive() = %#v, want %#v", got, want)
	}
}

func TestParseAdaptiveJSONNodes(t *testing.T) {
	got := parseAdaptive(`{"nodes":["192.0.2.1:443#US","198.51.100.2:443#JP"]}`)
	want := []string{"192.0.2.1:443#US", "198.51.100.2:443#JP"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseAdaptive(JSON) = %#v, want %#v", got, want)
	}
}

func TestRiskScore(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"0.4 (low)", 0.4},
		{"0.125", 0.125},
		{"", 0},
	}
	for _, test := range tests {
		if got := riskScore(test.input); got != test.want {
			t.Errorf("riskScore(%q) = %v, want %v", test.input, got, test.want)
		}
	}
}

func TestExtractCountryCode(t *testing.T) {
 tests := []struct{ label, want string }{
  {"US", "US"},
  {"USA", "US"},
  {"日本", "JP"},
 }
 for _, test := range tests {
  if got := extractCountryCode(test.label); got != test.want {
   t.Errorf("extractCountryCode(%q) = %q, want %q", test.label, got, test.want)
  }
 }
}

func TestWriteCacheSortsIPv4Numerically(t *testing.T) {
 path := t.TempDir() + "/cache.txt"
 writeCache(path, map[string]string{
  "192.0.2.10:443": "US",
  "192.0.2.2:443": "JP",
 })
 data, err := os.ReadFile(path)
 if err != nil { t.Fatal(err) }
 got := strings.Split(strings.TrimSpace(string(data)), "\n")
 want := []string{"192.0.2.2:443#JP", "192.0.2.10:443#US"}
 if !reflect.DeepEqual(got, want) {
  t.Fatalf("cache order = %#v, want %#v", got, want)
 }
}
