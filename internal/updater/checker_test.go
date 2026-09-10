package updater

import (
	"testing"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		cur    string
		latest string
		want   bool
	}{
		{"v2.4.0", "v2.5.0", true},
		{"v2.5.0", "v2.5.1", true},
		{"2.5.0", "v2.5.0", false},
		{"v2.5.1", "v2.4.9", false},
		{"v1.8.23", "v2.0.0", true},
		{"", "v2.0.0", true},
		{"v2.0.0", "", false},
	}

	for _, tt := range tests {
		got := IsNewer(tt.cur, tt.latest)
		if got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v; want %v", tt.cur, tt.latest, got, tt.want)
		}
	}
}
