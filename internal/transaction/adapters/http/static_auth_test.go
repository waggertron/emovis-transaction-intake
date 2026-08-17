package httpadapter

import "testing"

func TestStaticAPIKeysAuthenticatesConfiguredKeys(t *testing.T) {
	t.Parallel()

	auth := NewStaticAPIKeys(map[string]string{"partner-west": "secret-value"})
	partnerID, ok := auth.Authenticate("secret-value")
	if !ok || partnerID != "partner-west" {
		t.Fatalf("expected partner-west, got %q, %t", partnerID, ok)
	}
}

func TestStaticAPIKeysRejectsEmptyAndUnknownKeys(t *testing.T) {
	t.Parallel()

	auth := NewStaticAPIKeys(map[string]string{"partner-west": "secret-value"})
	for _, key := range []string{"", "wrong", "secret-value-extra"} {
		if partnerID, ok := auth.Authenticate(key); ok || partnerID != "" {
			t.Fatalf("expected %q to be rejected, got %q, %t", key, partnerID, ok)
		}
	}
}
