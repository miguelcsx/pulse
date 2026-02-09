package service

import (
	"testing"

	"github.com/pulse/stone/internal/model"
)

func TestBuildBridge_NoTags(t *testing.T) {
	got := buildBridge(nil, 0)
	want := "You share similar interests"
	if got != want {
		t.Fatalf("buildBridge() = %q, want %q", got, want)
	}
}

func TestBuildBridge_WithTags(t *testing.T) {
	tags := []model.Tag{{Name: "synthwave"}, {Name: "noir"}}
	got := buildBridge(tags, 2)
	want := "You both post about #synthwave and #noir"
	if got != want {
		t.Fatalf("buildBridge() = %q, want %q", got, want)
	}
}

func TestBuildBridge_WithMoreSharedTags(t *testing.T) {
	tags := []model.Tag{{Name: "synthwave"}, {Name: "noir"}, {Name: "glitch"}, {Name: "rain"}}
	got := buildBridge(tags, 6)
	want := "You both post about #synthwave, #noir and #glitch and 3 more shared tags"
	if got != want {
		t.Fatalf("buildBridge() = %q, want %q", got, want)
	}
}
