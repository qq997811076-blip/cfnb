package main

import (
	"encoding/json"
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
  {"USA123", "US"},
  {"USABC", ""},
  {"日本", "JP"},
  {"🇺🇸", "US"},
  {"🇺🇸🇨", ""},
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

func TestSyncSettingReadsShellAndPowerShell(t *testing.T) {
 shell := "github_token=\"token-value\"\ngithub_username=\"owner\"\nrepo_name=\"repo\"\nbranch=\"main\"\n"
 powershell := "$github_token = \"token-value\"\r\n$github_username = \"owner\"\r\n$repo_name = \"repo\"\r\n$branch = \"main\"\r\n"
 for _, source := range []string{shell, powershell} {
  for key, want := range map[string]string{"github_token": "token-value", "github_username": "owner", "repo_name": "repo", "branch": "main"} {
   if got := syncSetting(source, key); got != want {
    t.Errorf("syncSetting(%q) = %q, want %q", key, got, want)
   }
  }
 }
}

func TestLoadConfigUsesDefaultsWhenMissing(t *testing.T) {
 oldRoot, oldConfig := root, cfg
 root = t.TempDir()
 cfg = nil
 defer func() { root, cfg = oldRoot, oldConfig }()
 if err := loadConfig(); err != nil { t.Fatalf("loadConfig() error = %v", err) }
 if got := integer("GLOBAL_TOP_N"); got != 15 { t.Fatalf("GLOBAL_TOP_N = %d, want 15", got) }
 if got := str("OUTPUT_FILE"); got != "ip.txt" { t.Fatalf("OUTPUT_FILE = %q, want ip.txt", got) }
}

func TestSourceEnabledDefault(t *testing.T) {
 var defaultSource Source
 if err := json.Unmarshal([]byte(`{"url":"https://example.com"}`), &defaultSource); err != nil { t.Fatal(err) }
 if defaultSource.Enabled != nil { t.Fatalf("omitted enabled flag should stay nil, got %v", *defaultSource.Enabled) }
 var disabled Source
 if err := json.Unmarshal([]byte(`{"url":"https://example.com","enabled":false}`), &disabled); err != nil { t.Fatal(err) }
 if disabled.Enabled == nil || *disabled.Enabled { t.Fatalf("explicit false should be preserved, got %#v", disabled.Enabled) }
}
