package tasks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildPythonDancerCLIArgs_Default(t *testing.T) {
	args := buildPythonDancerCLIArgs("/videos/test.mp4", "/videos/test.funscript", false, defaultPythonDancerProfiles()[0])
	got := filepath.ToSlash(args[len(args)-1])
	if got != "/videos/test.mp4" {
		t.Fatalf("expected audio path at tail, got %q", got)
	}

	joined := filepath.ToSlash(args[0] + " " + args[1] + " " + args[2] + " " + args[3])
	if joined != "-m dancer.cli --cli --automap" {
		t.Fatalf("unexpected prefix args %v", args)
	}

	foundAutoSpeed := false
	for _, arg := range args {
		if arg == "--auto_speed" {
			foundAutoSpeed = true
		}
		if arg == "--heatmap" {
			t.Fatalf("did not expect --heatmap in PythonDancer task args: %v", args)
		}
		if arg == "--convert" {
			t.Fatalf("did not expect --convert in default args: %v", args)
		}
		if arg == "-y" || arg == "--yes" {
			t.Fatalf("did not expect overwrite flag in args: %v", args)
		}
	}
	if !foundAutoSpeed {
		t.Fatalf("expected tuning args in default args: %v", args)
	}
}

func TestBuildPythonDancerCLIArgs_WithConvert(t *testing.T) {
	args := buildPythonDancerCLIArgs("/videos/test.mp4", "/videos/test.funscript", true, defaultPythonDancerProfiles()[1])
	foundConvert := false
	foundNoPLP := false
	for _, arg := range args {
		if arg == "--convert" {
			foundConvert = true
		}
		if arg == "--no_plp" {
			foundNoPLP = true
		}
	}
	if !foundConvert {
		t.Fatalf("expected --convert in args: %v", args)
	}
	if foundNoPLP {
		t.Fatalf("did not expect --no_plp in gentle profile args: %v", args)
	}
}

func TestSiblingFunscriptPath(t *testing.T) {
	got := filepath.ToSlash(siblingFunscriptPath("/mnt/g/Videos/demo/video.mkv"))
	want := "/mnt/g/Videos/demo/video.funscript"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizePythonDancerBatchConcurrency(t *testing.T) {
	if got := normalizePythonDancerBatchConcurrency(0); got != 1 {
		t.Fatalf("expected default concurrency 1, got %d", got)
	}
	if got := normalizePythonDancerBatchConcurrency(99); got != 4 {
		t.Fatalf("expected max concurrency 4, got %d", got)
	}
}

func TestNormalizePythonDancerBatchLimit(t *testing.T) {
	if got := normalizePythonDancerBatchLimit(0); got != 0 {
		t.Fatalf("expected zero limit to mean all files, got %d", got)
	}
	if got := normalizePythonDancerBatchLimit(-1); got != 0 {
		t.Fatalf("expected negative limit to mean all files, got %d", got)
	}
	if got := normalizePythonDancerBatchLimit(999); got != 500 {
		t.Fatalf("expected max limit 500, got %d", got)
	}
}

func TestBuildPythonDancerCLIArgs_NoPLPProfile(t *testing.T) {
	args := buildPythonDancerCLIArgs("/videos/test.mp4", "/videos/test.funscript", false, defaultPythonDancerProfiles()[2])
	foundNoPLP := false
	for _, arg := range args {
		if arg == "--no_plp" {
			foundNoPLP = true
			break
		}
	}
	if !foundNoPLP {
		t.Fatalf("expected --no_plp in fallback profile args: %v", args)
	}
}

func TestAnalyzeGeneratedFunscriptFlagsRepetitiveFast(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "repetitive.funscript")

	type action struct {
		At  int64 `json:"at"`
		Pos int   `json:"pos"`
	}
	data := struct {
		Version string   `json:"version"`
		Range   int      `json:"range"`
		Actions []action `json:"actions"`
	}{
		Version: "1.0",
		Range:   100,
		Actions: []action{
			{At: 0, Pos: 86},
			{At: 20, Pos: 86},
			{At: 245, Pos: 85},
			{At: 469, Pos: 0},
			{At: 693, Pos: 100},
			{At: 917, Pos: 0},
			{At: 1152, Pos: 100},
			{At: 1386, Pos: 0},
			{At: 1610, Pos: 100},
			{At: 1834, Pos: 0},
			{At: 2058, Pos: 100},
			{At: 2282, Pos: 0},
			{At: 2517, Pos: 100},
			{At: 2752, Pos: 0},
			{At: 2976, Pos: 100},
			{At: 3200, Pos: 0},
		},
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := analyzeGeneratedFunscript(path)
	if err != nil {
		t.Fatal(err)
	}
	if !report.TooFast {
		t.Fatalf("expected TooFast=true, got %#v", report)
	}
	if !report.TooRepetitive {
		t.Fatalf("expected TooRepetitive=true, got %#v", report)
	}
	if !qualityNeedsRetry(report) {
		t.Fatalf("expected repetitive fast script to require post-processing, got %#v", report)
	}
}

func TestQualityNeedsRetrySkipsFastButGranularScript(t *testing.T) {
	report := funscriptQualityReport{
		TooFast:                 true,
		TooRepetitive:           false,
		ExtremeRatio:            0.71,
		AlternatingExtremeRatio: 0.69,
		UniquePositionCount:     59,
		MedianPosition:          27,
		ActionsPerSecond:        4.48,
	}
	if !qualityNeedsRetry(report) {
		t.Fatalf("expected off-center fast script to require post-processing, got %#v", report)
	}
}

func TestQualityNeedsRetrySkipsRepetitiveOnlyScript(t *testing.T) {
	report := funscriptQualityReport{
		TooFast:                 false,
		TooRepetitive:           true,
		ExtremeRatio:            0.95,
		AlternatingExtremeRatio: 0.93,
		UniquePositionCount:     6,
		ActionsPerSecond:        3.2,
	}
	if qualityNeedsRetry(report) {
		t.Fatalf("expected repetitive-only script to skip post-processing, got %#v", report)
	}
}

func TestQualityNeedsRetrySkipsCenteredFastExtremeScript(t *testing.T) {
	report := funscriptQualityReport{
		TooFast:                 true,
		TooRepetitive:           true,
		ExtremeRatio:            0.95,
		AlternatingExtremeRatio: 0.93,
		UniquePositionCount:     26,
		MedianPosition:          51,
		ActionsPerSecond:        4.44,
	}
	if qualityNeedsRetry(report) {
		t.Fatalf("expected centered fast script to skip post-processing, got %#v", report)
	}
}

func TestQualityNeedsRetryProcessesLowVarAlternatingPMVCase(t *testing.T) {
	report := funscriptQualityReport{
		TooFast:                 true,
		TooRepetitive:           true,
		ExtremeRatio:            0.99,
		AlternatingExtremeRatio: 0.99,
		UniquePositionCount:     7,
		MedianPosition:          72,
		ActionsPerSecond:        4.41,
	}
	if !qualityNeedsRetry(report) {
		t.Fatalf("expected low-var alternating PMV case to require post-processing, got %#v", report)
	}
}

func TestPostProcessGeneratedFunscriptSoftensRepetitiveScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "repetitive.funscript")

	type action struct {
		At  int64 `json:"at"`
		Pos int   `json:"pos"`
	}
	data := struct {
		Version string   `json:"version"`
		Range   int      `json:"range"`
		Actions []action `json:"actions"`
	}{
		Version: "1.0",
		Range:   100,
		Actions: []action{
			{At: 0, Pos: 100},
			{At: 120, Pos: 0},
			{At: 240, Pos: 100},
			{At: 360, Pos: 0},
			{At: 480, Pos: 100},
			{At: 600, Pos: 0},
			{At: 720, Pos: 100},
		},
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	before, err := analyzeGeneratedFunscript(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := postProcessGeneratedFunscript(path, before, postProcessModeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied {
		t.Fatalf("expected post-processing to apply, got %#v", result)
	}
	if result.After.ActionCount == 0 {
		t.Fatalf("expected non-empty post-processed output, got %#v", result.After)
	}
}

func TestDebinaryAlternatingExtremesKeepsRangeButAddsIntermediateLevels(t *testing.T) {
	actions := []Action{
		{At: 0, Pos: 85},
		{At: 224, Pos: 0},
		{At: 448, Pos: 100},
		{At: 672, Pos: 0},
		{At: 896, Pos: 100},
		{At: 1120, Pos: 0},
	}

	got := debinaryAlternatingExtremes(actions)
	if len(got) != len(actions) {
		t.Fatalf("expected %d actions, got %d", len(actions), len(got))
	}

	unique := map[int]struct{}{}
	for _, action := range got {
		unique[action.Pos] = struct{}{}
	}
	if len(unique) < 3 {
		t.Fatalf("expected more intermediate levels, got %#v", got)
	}
	if got[1].Pos != 50 {
		t.Fatalf("expected first de-binarized transition to pass through center, got %#v", got)
	}
	if got[2].Pos <= 5 || got[2].Pos >= 92 {
		t.Fatalf("expected first de-binarized valley to be intermediate, got %#v", got)
	}
	if got[4].Pos != 28 {
		t.Fatalf("expected extended 6-phase rebound, got %#v", got)
	}
	if got[5].Pos != 50 {
		t.Fatalf("expected 6-phase ramp to return through center, got %#v", got)
	}
}

func TestIsIgnorablePythonDancerError(t *testing.T) {
	if !isIgnorablePythonDancerError("ValueError: 'bboxes' cannot be empty") {
		t.Fatal("expected bbox error to be ignorable")
	}
	if isIgnorablePythonDancerError("some other failure") {
		t.Fatal("did not expect arbitrary error to be ignorable")
	}
}
