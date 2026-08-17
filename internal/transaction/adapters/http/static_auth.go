package httpadapter

import (
	"crypto/sha256"
	"crypto/subtle"
)

type apiKeyEntry struct {
	partnerID string
	digest    [sha256.Size]byte
}

type StaticAPIKeys struct {
	entries []apiKeyEntry
}

func NewStaticAPIKeys(keysByPartner map[string]string) *StaticAPIKeys {
	entries := make([]apiKeyEntry, 0, len(keysByPartner))
	for partnerID, key := range keysByPartner {
		if partnerID == "" || key == "" {
			continue
		}
		entries = append(entries, apiKeyEntry{partnerID: partnerID, digest: sha256.Sum256([]byte(key))})
	}
	return &StaticAPIKeys{entries: entries}
}

func (auth *StaticAPIKeys) Authenticate(apiKey string) (string, bool) {
	if apiKey == "" {
		return "", false
	}
	candidate := sha256.Sum256([]byte(apiKey))
	for _, entry := range auth.entries {
		if subtle.ConstantTimeCompare(candidate[:], entry.digest[:]) == 1 {
			return entry.partnerID, true
		}
	}
	return "", false
}
