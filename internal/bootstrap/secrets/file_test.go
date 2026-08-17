package secrets

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileProviderLoadsAndReloadsSecretValues(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	writeSecretFixture(t, path, `{"API_KEY":"first-key","PARTNER_ID":"partner-west"}`)
	provider := NewFileProvider(path)
	values, err := provider.Load(context.Background())
	if err != nil || values["API_KEY"] != "first-key" || values["PARTNER_ID"] != "partner-west" {
		t.Fatalf("initial load: %#v %v", values, err)
	}
	writeSecretFixture(t, path, `{"API_KEY":"rotated-key","PARTNER_ID":"partner-west"}`)
	values, err = provider.Load(context.Background())
	if err != nil || values["API_KEY"] != "rotated-key" {
		t.Fatalf("rotated load: %#v %v", values, err)
	}
}

func TestFileProviderRejectsAbsentMalformedEmptyAndOversizedDataWithoutLeaking(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	for _, test := range []struct {
		name    string
		path    string
		content string
	}{
		{name: "absent", path: filepath.Join(directory, "absent.json")},
		{name: "malformed", path: filepath.Join(directory, "malformed.json"), content: `{"API_KEY":"sensitive-value"`},
		{name: "empty value", path: filepath.Join(directory, "empty.json"), content: `{"API_KEY":""}`},
		{name: "oversized", path: filepath.Join(directory, "large.json"), content: `{"API_KEY":"` + strings.Repeat("x", MaxFileBytes) + `"}`},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if test.content != "" {
				writeSecretFixture(t, test.path, test.content)
			}
			_, err := NewFileProvider(test.path).Load(context.Background())
			if err == nil || strings.Contains(err.Error(), "sensitive-value") {
				t.Fatalf("expected redacted rejection, got %v", err)
			}
		})
	}
}

func TestLookupUsesSecretValuesWithoutOverridingExplicitEnvironment(t *testing.T) {
	t.Parallel()
	lookup := Overlay(func(key string) string {
		if key == "API_KEY" {
			return "explicit"
		}
		return ""
	}, map[string]string{"API_KEY": "provider", "PARTNER_ID": "partner-west"})
	if lookup("API_KEY") != "explicit" || lookup("PARTNER_ID") != "partner-west" || lookup("UNKNOWN") != "" {
		t.Fatal("unexpected secret overlay precedence")
	}
}

func writeSecretFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
