package service

import "testing"

func TestNormalizeDemoHandle(t *testing.T) {
	got, err := normalizeDemoHandle(" @Demo_User-1 ")
	if err != nil {
		t.Fatalf("expected valid demo handle, got error %v", err)
	}
	if got != "demo_user-1" {
		t.Fatalf("unexpected normalized handle %q", got)
	}
}

func TestNormalizeDemoHandleRejectsInvalid(t *testing.T) {
	invalid := []string{"ab", "demo user", "demo/user", "ñandú"}
	for _, raw := range invalid {
		if _, err := normalizeDemoHandle(raw); err == nil {
			t.Fatalf("expected %q to be invalid", raw)
		}
	}
}
