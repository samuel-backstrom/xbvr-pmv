package tasks

import (
	"encoding/json"
	"testing"
)

func TestNormalizePMVImportBatchLimit(t *testing.T) {
	if got := normalizePMVImportBatchLimit(0); got != 0 {
		t.Fatalf("expected zero to mean all, got %d", got)
	}
	if got := normalizePMVImportBatchLimit(-1); got != 0 {
		t.Fatalf("expected negative to mean all, got %d", got)
	}
	if got := normalizePMVImportBatchLimit(999); got != maxPMVImportBatchLimit {
		t.Fatalf("expected capped limit %d, got %d", maxPMVImportBatchLimit, got)
	}
	if got := normalizePMVImportBatchLimit(7); got != 7 {
		t.Fatalf("expected passthrough limit 7, got %d", got)
	}
}

func TestNormalizePMVImportBatchConcurrency(t *testing.T) {
	if got := normalizePMVImportBatchConcurrency(0); got != defaultPMVImportBatchConcurrency {
		t.Fatalf("expected default concurrency %d, got %d", defaultPMVImportBatchConcurrency, got)
	}
	if got := normalizePMVImportBatchConcurrency(-1); got != defaultPMVImportBatchConcurrency {
		t.Fatalf("expected default concurrency %d, got %d", defaultPMVImportBatchConcurrency, got)
	}
	if got := normalizePMVImportBatchConcurrency(999); got != maxPMVImportBatchConcurrency {
		t.Fatalf("expected capped concurrency %d, got %d", maxPMVImportBatchConcurrency, got)
	}
	if got := normalizePMVImportBatchConcurrency(5); got != 5 {
		t.Fatalf("expected passthrough concurrency 5, got %d", got)
	}
}

func TestConvertPMVHavenScriptDataToFunscriptFromCSV(t *testing.T) {
	raw := []byte("#Created by Handy SDK v2\n200,5\n0,100\n100,0\n")

	got, err := convertPMVHavenScriptDataToFunscript(raw, 123.4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var script Script
	if err := json.Unmarshal(got, &script); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if script.Version != "1.0" {
		t.Fatalf("expected version 1.0, got %#v", script.Version)
	}
	if script.Range != 100 {
		t.Fatalf("expected range 100, got %d", script.Range)
	}
	if script.Metadata == nil || script.Metadata.Duration != 123 {
		t.Fatalf("expected metadata duration 123, got %#v", script.Metadata)
	}
	if len(script.Actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(script.Actions))
	}
	if script.Actions[0].At != 0 || script.Actions[0].Pos != 100 {
		t.Fatalf("expected first action sorted to 0,100; got %#v", script.Actions[0])
	}
	if script.Actions[2].At != 200 || script.Actions[2].Pos != 5 {
		t.Fatalf("expected last action 200,5; got %#v", script.Actions[2])
	}
}
