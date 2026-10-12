package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		{"prefix 0.4 (low)", 0},
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

func TestApplyCachedTagsPreservesFullLabel(t *testing.T) {
 nodes := []string{"192.0.2.1:443#US"}
 applyCachedTags(nodes, map[string]string{"192.0.2.1:443": "US California Los Angeles AS123 Example ISP"})
 want := []string{"192.0.2.1:443#US California Los Angeles AS123 Example ISP"}
 if !reflect.DeepEqual(nodes, want) {
  t.Fatalf("applyCachedTags() = %#v, want %#v", nodes, want)
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


func TestWriteIPPreservesConfiguredFormatting(t *testing.T) {
	oldRoot, oldConfig := root, cfg
	root = t.TempDir()
	cfg = Config{
		"OUTPUT_FILE": "ip.txt",
		"AD_HEADER_ENABLED": true,
		"AD_HEADER_LINES": []any{"# header"},
		"AD_FOOTER_ENABLED": true,
		"AD_FOOTER_LINES": []any{"# footer"},
		"AD_PERLINE_ENABLED": true,
		"AD_PERLINE_TEXT": " | checked",
		"IP_TXT_SHOW_BANDWIDTH": true,
		"IP_TXT_SHOW_HTTP_LATENCY": true,
		"IP_TXT_SHOW_HTTP_JITTER": true,
		"IP_TXT_SHOW_LATENCY": true,
	}
	t.Cleanup(func() { root, cfg = oldRoot, oldConfig })

	node := "192.0.2.1:443#US"
	err := writeIP(
		[]string{node},
		map[string]float64{node: 12.34},
		map[string]float64{node: 0.012},
		map[string]float64{node: 20.5},
		map[string]float64{node: 1.25},
	)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "ip.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := "# header\n192.0.2.1:443#US 12.34 Mbps 20.50 ms 1.25 ms 12.00 ms | checked\n# footer\n"
	if string(data) != want {
		t.Fatalf("output = %q, want %q", string(data), want)
	}
}


func TestRunPoolPreservesInputOrder(t *testing.T) {
	items := []string{"first", "second", "third", "fourth"}
	got := runPool(items, 3, func(item string) string {
		return item + "-done"
	}, "test")
	want := []string{"first-done", "second-done", "third-done", "fourth-done"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runPool() = %#v, want %#v", got, want)
	}
}

func TestRunPoolHandlesEmptyInput(t *testing.T) {
	got := runPool([]string{}, 4, func(item string) string {
		return item
	}, "empty-test")
	if len(got) != 0 {
		t.Fatalf("runPool(empty) returned %#v, want empty result", got)
	}
}
