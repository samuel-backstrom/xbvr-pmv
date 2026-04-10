package tasks

import "testing"

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
