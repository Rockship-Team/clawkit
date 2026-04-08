package oauth

import (
	"testing"
)

func TestProvidersRegistered(t *testing.T) {
	expected := []string{"zalo_personal", "zalo_oa", "google", "facebook"}

	for _, name := range expected {
		p, err := Get(name)
		if err != nil {
			t.Errorf("provider %s not registered: %v", name, err)
			continue
		}
		if p.Name() != name {
			t.Errorf("provider name mismatch: got %s, want %s", p.Name(), name)
		}
		if p.Display() == "" {
			t.Errorf("provider %s has empty display name", name)
		}
	}
}

func TestGetUnknownProvider(t *testing.T) {
	_, err := Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestListProviders(t *testing.T) {
	providers := ListProviders()
	if len(providers) < 4 {
		t.Errorf("expected at least 4 providers, got %d: %v", len(providers), providers)
	}
}
