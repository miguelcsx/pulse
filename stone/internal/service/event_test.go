package service

import "testing"

func TestImplicitSignalFromEvent_View(t *testing.T) {
	got := implicitSignalFromEvent("view", `{"dwell_ms": 5000}`)
	if got <= 1.0 {
		t.Fatalf("expected view signal > 1.0, got %f", got)
	}
}

func TestImplicitSignalFromEvent_Skip(t *testing.T) {
	got := implicitSignalFromEvent("skip", `{"at_ms": 200}`)
	if got >= 0 {
		t.Fatalf("expected negative skip signal, got %f", got)
	}
}

func TestImplicitSignalFromEvent_Unknown(t *testing.T) {
	got := implicitSignalFromEvent("something_else", `{}`)
	if got != 0 {
		t.Fatalf("expected zero for unknown event, got %f", got)
	}
}

func TestNormalizeEventContract(t *testing.T) {
	eventType, targetType, err := normalizeEventContract(" View ", " Content ")
	if err != nil {
		t.Fatalf("expected valid event contract, got error %v", err)
	}
	if eventType != "view" || targetType != "content" {
		t.Fatalf("unexpected normalized values %q/%q", eventType, targetType)
	}
}

func TestNormalizeEventContract_InvalidType(t *testing.T) {
	if _, _, err := normalizeEventContract("weird", "content"); err == nil {
		t.Fatalf("expected error for unsupported event type")
	}
}

func TestNormalizeMetadata(t *testing.T) {
	got, err := normalizeMetadata(`{"dwell_ms":1200}`)
	if err != nil {
		t.Fatalf("expected valid metadata, got error %v", err)
	}
	if got != `{"dwell_ms":1200}` {
		t.Fatalf("unexpected metadata normalization result: %s", got)
	}
}

func TestNormalizeMetadata_InvalidShape(t *testing.T) {
	if _, err := normalizeMetadata(`[]`); err == nil {
		t.Fatalf("expected error for non-object metadata")
	}
}
