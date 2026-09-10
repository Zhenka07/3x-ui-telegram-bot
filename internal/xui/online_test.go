package xui

import (
	"encoding/json"
	"testing"
)

func TestExtractOnlineEmails_DirectArray(t *testing.T) {
	raw := json.RawMessage(`["user1", "user2", "user1"]`)
	emails := extractOnlineEmails(raw)
	if len(emails) != 2 {
		t.Fatalf("expected 2 unique emails, got %d: %v", len(emails), emails)
	}
	if emails[0] != "user1" || emails[1] != "user2" {
		t.Errorf("unexpected emails: %v", emails)
	}
}

func TestExtractOnlineEmails_Object(t *testing.T) {
	raw := json.RawMessage(`{"success": true, "obj": ["userA", "userB"]}`)
	emails := extractOnlineEmails(raw)
	if len(emails) != 2 {
		t.Fatalf("expected 2 emails, got %d: %v", len(emails), emails)
	}
	if emails[0] != "userA" || emails[1] != "userB" {
		t.Errorf("unexpected emails: %v", emails)
	}
}
