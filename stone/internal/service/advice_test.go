package service

import (
	"testing"

	"github.com/pulse/stone/internal/model"
)

func TestBuildAdviceBridgeReasonMentor(t *testing.T) {
	ask := model.Ask{Topic: "first customers"}
	profile := model.TrustProfile{LivedExperience: "Built a SaaS from zero to first revenue."}

	got := buildAdviceBridgeReason(ask, profile, model.BridgeTypeMentor)
	want := "They have lived experience related to first customers and can give you practical perspective."
	if got != want {
		t.Fatalf("buildAdviceBridgeReason() = %q, want %q", got, want)
	}
}

func TestInferTopicFromQuestion(t *testing.T) {
	got := inferTopic("I am launching and need my first 10 customers")
	if got != "customers" {
		t.Fatalf("inferTopic() = %q, want customers", got)
	}
}

func TestOverlapScoreHasSharedContext(t *testing.T) {
	got := overlapScore("first customers launch", "customers go to market launch")
	if got <= 0 {
		t.Fatalf("overlapScore() = %f, want positive score", got)
	}
}

func TestCommonsResponsesConvertsNilToEmptySlice(t *testing.T) {
	got := commonsResponses(nil)
	if got == nil {
		t.Fatal("commonsResponses(nil) returned nil, want empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("commonsResponses(nil) length = %d, want 0", len(got))
	}
}
