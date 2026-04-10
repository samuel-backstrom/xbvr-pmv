package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/djherbis/times"
	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

type PythonDancerBatchRequest struct {
	Limit           int    `json:"limit"`
	Concurrency     int    `json:"concurrency,omitempty"`
	VolumeID        uint   `json:"volume_id,omitempty"`
	PathPrefix      string `json:"path_prefix,omitempty"`
	FileID          uint   `json:"file_id,omitempty"`
	ForceRegenerate bool   `json:"force_regenerate,omitempty"`
	PostProcessMode string `json:"post_process_mode,omitempty"`
}

type PythonDancerBatchItem struct {
	FileID      uint   `json:"file_id"`
	Filename    string `json:"filename"`
	OutputPath  string `json:"output_path,omitempty"`
	SceneID     string `json:"scene_id,omitempty"`
	UsedConvert bool   `json:"used_convert,omitempty"`
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
}

type PythonDancerBatchResult struct {
	Scanned         int                     `json:"scanned"`
	Generated       int                     `json:"generated"`
	Matched         int                     `json:"matched"`
	SkippedExisting int                     `json:"skipped_existing"`
	Errors          int                     `json:"errors"`
	Results         []PythonDancerBatchItem `json:"results"`
}

type pythonExecSpec struct {
	Name       string
	ArgsPrefix []string
}

type pythonDancerTuningProfile struct {
	Name        string
	AutoPitch   int
	AutoSpeed   int
	AutoPer     int
	NoPLP       bool
	Description string
}

type funscriptQualityReport struct {
	ActionCount             int
	DurationMs              int64
	MedianDeltaMs           int64
	MedianPosition          int
	ActionsPerSecond        float64
	MedianStrokeUnitsPerSec float64
	ExtremeRatio            float64
	AlternatingExtremeRatio float64
	UniquePositionCount     int
	TooFast                 bool
	TooRepetitive           bool
}

type postProcessResult struct {
	Applied bool
	Before  funscriptQualityReport
	After   funscriptQualityReport
}

const (
	postProcessModeAuto   = "auto"
	postProcessModeAlways = "always"
	postProcessModeNever  = "never"
)

var runPythonDancerCommand = func(ctx context.Context, cmd *exec.Cmd) (string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(strings.Join([]string{
		stdout.String(),
		stderr.String(),
	}, "\n"))
	if err != nil {
		if out != "" {
			return out, fmt.Errorf("%w: %s", err, out)
		}
		return out, err
	}
	return out, nil
}

func normalizePythonDancerBatchLimit(limit int) int {
	if limit <= 0 {
		return 0
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func normalizePythonDancerBatchConcurrency(concurrency int) int {
	if concurrency <= 0 {
		return 1
	}
	if concurrency > 4 {
		return 4
	}
	return concurrency
}

func normalizePostProcessMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case postProcessModeAlways:
		return postProcessModeAlways
	case postProcessModeNever:
		return postProcessModeNever
	default:
		return postProcessModeAuto
	}
}

func findPythonDancerDir() (string, error) {
	if envDir := strings.TrimSpace(os.Getenv("PYTHONDANCER_DIR")); envDir != "" {
		if pathLooksLikePythonDancer(envDir) {
			return envDir, nil
		}
	}

	candidates := make([]string, 0, 5)
	if wd, err := os.Getwd(); err == nil && wd != "" {
		candidates = append(candidates, wd)
	}
	if exe, err := os.Executable(); err == nil && exe != "" {
		candidates = append(candidates, filepath.Dir(exe))
	}
	if common.AppDir != "" {
		candidates = append(candidates, common.AppDir)
	}
	if _, callerFile, _, ok := runtime.Caller(0); ok && callerFile != "" {
		candidates = append(candidates, filepath.Dir(callerFile))
	}

	seen := map[string]bool{}
	for _, base := range candidates {
		cur := filepath.Clean(base)
		for i := 0; i < 6; i++ {
			if cur == "" || seen[cur] {
				break
			}
			seen[cur] = true
			direct := filepath.Join(cur, "PythonDancer")
			if pathLooksLikePythonDancer(direct) {
				return direct, nil
			}
			if pathLooksLikePythonDancer(cur) {
				return cur, nil
			}
			parent := filepath.Dir(cur)
			if parent == cur {
				break
			}
			cur = parent
		}
	}

	return "", fmt.Errorf("PythonDancer directory not found")
}

func pathLooksLikePythonDancer(dir string) bool {
	if dir == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "dancer", "cli.py")); err != nil {
		return false
	}
	return true
}

func findPythonExec(pythonDancerDir string) (pythonExecSpec, error) {
	if pythonDancerDir != "" {
		venvCandidates := []string{
			filepath.Join(pythonDancerDir, ".venv", "bin", "python"),
			filepath.Join(pythonDancerDir, ".venv", "Scripts", "python.exe"),
		}
		for _, candidate := range venvCandidates {
			if _, err := os.Stat(candidate); err == nil {
				return pythonExecSpec{Name: candidate}, nil
			}
		}
	}

	candidates := make([]pythonExecSpec, 0, 3)
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			pythonExecSpec{Name: "py", ArgsPrefix: []string{"-3"}},
			pythonExecSpec{Name: "python"},
			pythonExecSpec{Name: "python3"},
		)
	} else {
		candidates = append(candidates,
			pythonExecSpec{Name: "python3"},
			pythonExecSpec{Name: "python"},
		)
	}

	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate.Name); err == nil {
			return candidate, nil
		}
	}

	return pythonExecSpec{}, fmt.Errorf("no Python executable found for PythonDancer")
}

func siblingFunscriptPath(videoPath string) string {
	return strings.TrimSuffix(videoPath, filepath.Ext(videoPath)) + ".funscript"
}

func siblingHeatmapPath(scriptPath string) string {
	ext := filepath.Ext(scriptPath)
	base := strings.TrimSuffix(scriptPath, ext)
	return base + "_heatmap.png"
}

func fileExistsNonEmpty(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

func defaultPythonDancerProfiles() []pythonDancerTuningProfile {
	return []pythonDancerTuningProfile{
		{
			Name:        "default",
			AutoPitch:   20,
			AutoSpeed:   250,
			AutoPer:     65,
			Description: "default automap profile",
		},
		{
			Name:        "gentle",
			AutoPitch:   28,
			AutoSpeed:   170,
			AutoPer:     45,
			Description: "lower energy target to reduce repetitive extremes",
		},
		{
			Name:        "gentle-no-plp",
			AutoPitch:   35,
			AutoSpeed:   120,
			AutoPer:     30,
			NoPLP:       true,
			Description: "fallback with slower target and PLP disabled",
		},
	}
}

func buildPythonDancerCLIArgs(videoPath, outputPath string, convert bool, profile pythonDancerTuningProfile) []string {
	args := []string{
		"-m", "dancer.cli",
		"--cli",
		"--automap",
		"--auto_pitch", fmt.Sprintf("%d", profile.AutoPitch),
		"--auto_speed", fmt.Sprintf("%d", profile.AutoSpeed),
		"--auto_per", fmt.Sprintf("%d", profile.AutoPer),
		"--out_path", outputPath,
	}
	if profile.NoPLP {
		args = append(args, "--no_plp")
	}
	if convert {
		args = append(args, "--convert")
	}
	args = append(args, videoPath)
	return args
}

func pythonPathEnvValue(pythonDancerDir string) string {
	current := strings.TrimSpace(os.Getenv("PYTHONPATH"))
	if current == "" {
		return pythonDancerDir
	}
	return pythonDancerDir + string(os.PathListSeparator) + current
}

func matplotlibConfigDir(pythonDancerDir string) string {
	return filepath.Join(pythonDancerDir, ".cache", "matplotlib")
}

func commandEnvWithPythonDancer(pythonDancerDir string) []string {
	env := os.Environ()
	cacheDir := matplotlibConfigDir(pythonDancerDir)
	_ = os.MkdirAll(cacheDir, 0o755)
	env = append(env, "PYTHONPATH="+pythonPathEnvValue(pythonDancerDir))
	env = append(env, "MPLCONFIGDIR="+cacheDir)
	env = append(env, "PATH="+common.BinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return env
}

func removeGeneratedArtifacts(scriptPath string) {
	_ = os.Remove(scriptPath)
	_ = os.Remove(siblingHeatmapPath(scriptPath))
}

func isIgnorablePythonDancerError(output string) bool {
	return strings.Contains(output, "ValueError: 'bboxes' cannot be empty")
}

func appendOutputNote(output, note string) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return strings.TrimSpace(output)
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return note
	}
	return output + "\n" + note
}

func qualityNeedsRetry(report funscriptQualityReport) bool {
	if !report.TooFast {
		return false
	}
	if report.UniquePositionCount <= 8 && report.AlternatingExtremeRatio >= 0.85 {
		return true
	}
	if report.ExtremeRatio >= 0.65 && report.AlternatingExtremeRatio >= 0.60 &&
		(report.MedianPosition <= 35 || report.MedianPosition >= 65) {
		return true
	}
	return false
}

func formatQualityReport(report funscriptQualityReport) string {
	issues := make([]string, 0, 2)
	if report.TooFast {
		issues = append(issues, "too_fast")
	}
	if report.TooRepetitive {
		issues = append(issues, "too_repetitive")
	}
	if len(issues) == 0 {
		issues = append(issues, "ok")
	}

	return fmt.Sprintf(
		"quality=%s actions=%d duration_ms=%d median_delta_ms=%d median_pos=%d aps=%.2f median_speed=%.1f extreme_ratio=%.2f alternating_extreme_ratio=%.2f unique_positions=%d",
		strings.Join(issues, ","),
		report.ActionCount,
		report.DurationMs,
		report.MedianDeltaMs,
		report.MedianPosition,
		report.ActionsPerSecond,
		report.MedianStrokeUnitsPerSec,
		report.ExtremeRatio,
		report.AlternatingExtremeRatio,
		report.UniquePositionCount,
	)
}

func analyzeGeneratedFunscript(path string) (funscriptQualityReport, error) {
	script, err := LoadFunscriptData(path)
	if err != nil {
		return funscriptQualityReport{}, err
	}
	if len(script.Actions) < 2 {
		return funscriptQualityReport{
			ActionCount: len(script.Actions),
		}, nil
	}

	deltas := make([]int64, 0, len(script.Actions)-1)
	speeds := make([]float64, 0, len(script.Actions)-1)
	uniquePositions := map[int]struct{}{}
	extremes := 0
	alternatingExtremes := 0
	for i, action := range script.Actions {
		uniquePositions[action.Pos] = struct{}{}
		if action.Pos <= 2 || action.Pos >= 98 {
			extremes++
		}
		if i == 0 {
			continue
		}
		prev := script.Actions[i-1]
		delta := action.At - prev.At
		if delta <= 0 {
			continue
		}
		deltas = append(deltas, delta)
		speeds = append(speeds, math.Abs(float64(action.Pos-prev.Pos))*1000.0/float64(delta))
		if (prev.Pos <= 2 || prev.Pos >= 98) && (action.Pos <= 2 || action.Pos >= 98) && prev.Pos != action.Pos {
			alternatingExtremes++
		}
	}

	if len(deltas) == 0 {
		return funscriptQualityReport{
			ActionCount:         len(script.Actions),
			DurationMs:          script.Actions[len(script.Actions)-1].At,
			UniquePositionCount: len(uniquePositions),
		}, nil
	}

	sort.Slice(deltas, func(i, j int) bool { return deltas[i] < deltas[j] })
	sort.Float64s(speeds)

	durationMs := script.Actions[len(script.Actions)-1].At - script.Actions[0].At
	actionsPerSecond := 0.0
	if durationMs > 0 {
		actionsPerSecond = float64(len(script.Actions)-1) * 1000.0 / float64(durationMs)
	}

	report := funscriptQualityReport{
		ActionCount:             len(script.Actions),
		DurationMs:              durationMs,
		MedianDeltaMs:           deltas[len(deltas)/2],
		MedianPosition:          positionPercentile(script.Actions, 0.50),
		ActionsPerSecond:        actionsPerSecond,
		MedianStrokeUnitsPerSec: speeds[len(speeds)/2],
		ExtremeRatio:            float64(extremes) / float64(len(script.Actions)),
		AlternatingExtremeRatio: float64(alternatingExtremes) / float64(len(deltas)),
		UniquePositionCount:     len(uniquePositions),
	}

	report.TooFast = report.MedianStrokeUnitsPerSec >= 360 || report.MedianDeltaMs <= 150 || report.ActionsPerSecond >= 5.5
	report.TooRepetitive = (report.ExtremeRatio >= 0.80 && report.AlternatingExtremeRatio >= 0.70) || (report.UniquePositionCount <= 6 && report.AlternatingExtremeRatio >= 0.60)
	return report, nil
}

func positionPercentile(actions []Action, q float64) int {
	if len(actions) == 0 {
		return 50
	}
	positions := make([]int, len(actions))
	for i, action := range actions {
		positions[i] = action.Pos
	}
	sort.Ints(positions)
	if len(positions) == 1 {
		return positions[0]
	}
	idx := int(math.Round(float64(len(positions)-1) * q))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(positions) {
		idx = len(positions) - 1
	}
	return positions[idx]
}

func clampActionPos(pos int) int {
	if pos < 0 {
		return 0
	}
	if pos > 100 {
		return 100
	}
	return pos
}

func interpolateActionPos(actions []Action, at int64) float64 {
	if len(actions) == 0 {
		return 50.0
	}
	if at <= actions[0].At {
		return float64(actions[0].Pos)
	}
	for i := 1; i < len(actions); i++ {
		if at > actions[i].At {
			continue
		}
		prev := actions[i-1]
		cur := actions[i]
		if cur.At == prev.At {
			return float64(cur.Pos)
		}
		progress := float64(at-prev.At) / float64(cur.At-prev.At)
		return float64(prev.Pos) + (float64(cur.Pos)-float64(prev.Pos))*progress
	}
	return float64(actions[len(actions)-1].Pos)
}

func targetResampleStep(report funscriptQualityReport) int64 {
	target := report.MedianDeltaMs
	if target < 220 {
		target = 220
	}
	if report.TooFast && target < 240 {
		target = 240
	}
	if report.ActionsPerSecond >= 4.5 && target < 208 {
		target = 208
	}
	if report.AlternatingExtremeRatio >= 0.90 && target < 208 {
		target = 208
	}
	return target
}

func amplitudeCompressionFactor(report funscriptQualityReport) float64 {
	factor := 1.0
	if report.TooRepetitive {
		factor = 0.92
	}
	if report.ExtremeRatio >= 0.95 {
		factor = 0.85
	}
	return factor
}

func smoothActionPositions(actions []Action) []Action {
	if len(actions) < 3 {
		return actions
	}
	out := make([]Action, len(actions))
	copy(out, actions)
	for i := 1; i < len(actions)-1; i++ {
		smoothed := int(math.Round((float64(actions[i-1].Pos) + 2*float64(actions[i].Pos) + float64(actions[i+1].Pos)) / 4.0))
		out[i].Pos = clampActionPos(smoothed)
	}
	return out
}

func lowPassActionPositions(actions []Action, alpha float64) []Action {
	if len(actions) == 0 {
		return actions
	}
	if alpha <= 0 {
		alpha = 0.35
	}
	if alpha > 1 {
		alpha = 1
	}

	out := make([]Action, len(actions))
	copy(out, actions)
	filtered := float64(actions[0].Pos)
	out[0].Pos = clampActionPos(int(math.Round(filtered)))
	for i := 1; i < len(actions); i++ {
		filtered = filtered + alpha*(float64(actions[i].Pos)-filtered)
		out[i].Pos = clampActionPos(int(math.Round(filtered)))
	}
	return out
}

func softenExtremeTarget(pos int) int {
	switch {
	case pos <= 2:
		return 10
	case pos >= 98:
		return 90
	default:
		return pos
	}
}

func isAlternatingExtremePair(a, b int) bool {
	return (a <= 2 && b >= 98) || (a >= 98 && b <= 2)
}

func isExtremePos(pos int) bool {
	return pos <= 2 || pos >= 98
}

func debinaryAlternatingExtremes(actions []Action) []Action {
	if len(actions) < 2 {
		return actions
	}

	out := make([]Action, len(actions))
	copy(out, actions)

	low := 8
	lowMid := 28
	center := 50
	highMid := 72
	high := 92
	lowSeq := []int{low, lowMid, center, highMid, high, center}
	highSeq := []int{high, highMid, center, lowMid, low, center}

	for i := 0; i < len(actions); {
		if !isExtremePos(actions[i].Pos) {
			out[i].Pos = actions[i].Pos
			i++
			continue
		}

		runEnd := i
		for runEnd+1 < len(actions) && isAlternatingExtremePair(actions[runEnd].Pos, actions[runEnd+1].Pos) {
			runEnd++
		}

		if runEnd == i {
			out[i].Pos = softenExtremeTarget(actions[i].Pos)
			i++
			continue
		}

		seq := lowSeq
		if actions[i].Pos >= 98 {
			seq = highSeq
		}
		if i > 0 {
			prev := out[i-1].Pos
			if actions[i].Pos <= 2 && prev > center {
				seq = []int{center, lowMid, low, lowMid, center, highMid}
			}
			if actions[i].Pos >= 98 && prev < center {
				seq = []int{center, highMid, high, highMid, center, lowMid}
			}
		}

		phase := 0
		for j := i; j <= runEnd; j++ {
			out[j].Pos = seq[phase]
			phase = (phase + 1) % len(seq)
		}
		i = runEnd + 1
	}

	return out
}

func actionPositionStats(actions []Action) (minPos, maxPos, p10, p90, median int) {
	if len(actions) == 0 {
		return 50, 50, 50, 50, 50
	}
	positions := make([]int, len(actions))
	for i, action := range actions {
		positions[i] = action.Pos
	}
	sort.Ints(positions)
	idx := func(q float64) int {
		if len(positions) == 1 {
			return 0
		}
		return int(math.Round(float64(len(positions)-1) * q))
	}
	return positions[0], positions[len(positions)-1], positions[idx(0.10)], positions[idx(0.90)], positions[idx(0.50)]
}

func expandActionRange(actions []Action, report funscriptQualityReport) []Action {
	if len(actions) < 2 {
		return actions
	}
	_, _, p10, p90, median := actionPositionStats(actions)
	currentSpan := p90 - p10
	targetSpan := 72
	if report.ExtremeRatio >= 0.95 || report.AlternatingExtremeRatio >= 0.90 {
		targetSpan = 78
	}
	if currentSpan >= targetSpan {
		return actions
	}
	if currentSpan < 6 {
		currentSpan = 6
	}

	scale := float64(targetSpan) / float64(currentSpan)
	center := float64(median)
	if center < 35 {
		center = 35
	}
	if center > 65 {
		center = 65
	}

	out := make([]Action, len(actions))
	copy(out, actions)
	for i, action := range actions {
		expanded := center + (float64(action.Pos)-center)*scale
		out[i].Pos = clampActionPos(int(math.Round(expanded)))
	}
	return out
}

func dedupeActions(actions []Action, minDelta int64) []Action {
	if len(actions) == 0 {
		return actions
	}
	out := make([]Action, 0, len(actions))
	out = append(out, actions[0])
	for i := 1; i < len(actions); i++ {
		last := out[len(out)-1]
		cur := actions[i]
		if cur.At-last.At < minDelta && math.Abs(float64(cur.Pos-last.Pos)) < 4 {
			continue
		}
		if cur.Pos == last.Pos && cur.At-last.At < minDelta*2 {
			out[len(out)-1] = cur
			continue
		}
		out = append(out, cur)
	}
	if len(out) == 1 && len(actions) > 1 {
		out = append(out, actions[len(actions)-1])
	}
	return out
}

func writeFunscriptData(path string, script Script) error {
	data, err := json.Marshal(script)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func postProcessGeneratedFunscript(path string, report funscriptQualityReport, mode string) (postProcessResult, error) {
	result := postProcessResult{Before: report}
	mode = normalizePostProcessMode(mode)
	shouldProcess := mode == postProcessModeAlways || (mode == postProcessModeAuto && qualityNeedsRetry(report))
	if !shouldProcess {
		return result, nil
	}

	script, err := LoadFunscriptData(path)
	if err != nil {
		return result, err
	}
	if len(script.Actions) < 3 {
		return result, nil
	}

	var processed []Action
	if report.AlternatingExtremeRatio >= 0.90 && report.ExtremeRatio >= 0.90 {
		processed = debinaryAlternatingExtremes(script.Actions)
		processed = expandActionRange(processed, report)
		processed = dedupeActions(processed, 40)
	} else {
		step := targetResampleStep(report)
		compression := amplitudeCompressionFactor(report)
		startAt := script.Actions[0].At
		endAt := script.Actions[len(script.Actions)-1].At
		if endAt <= startAt {
			return result, nil
		}

		processed = make([]Action, 0, len(script.Actions))
		for at := startAt; at <= endAt; at += step {
			pos := interpolateActionPos(script.Actions, at)
			pos = 50.0 + (pos-50.0)*compression
			processed = append(processed, Action{
				At:  at,
				Pos: clampActionPos(int(math.Round(pos))),
			})
		}
		if processed[len(processed)-1].At != endAt {
			pos := interpolateActionPos(script.Actions, endAt)
			pos = 50.0 + (pos-50.0)*compression
			processed = append(processed, Action{
				At:  endAt,
				Pos: clampActionPos(int(math.Round(pos))),
			})
		}

		alpha := 0.45
		if report.TooRepetitive {
			alpha = 0.40
		}
		processed = lowPassActionPositions(processed, alpha)
		if report.TooRepetitive && report.AlternatingExtremeRatio >= 0.85 {
			processed = smoothActionPositions(processed)
		}
		processed = expandActionRange(processed, report)
		processed = dedupeActions(processed, maxInt64(step/3, 72))
	}

	if len(processed) < 2 {
		return result, nil
	}

	sort.Slice(processed, func(i, j int) bool { return processed[i].At < processed[j].At })
	script.Actions = processed
	if script.Metadata != nil {
		script.Metadata.Duration = script.Actions[len(script.Actions)-1].At / 1000
	}
	if err := writeFunscriptData(path, script); err != nil {
		return result, err
	}

	after, err := analyzeGeneratedFunscript(path)
	if err != nil {
		return result, err
	}
	result.Applied = true
	result.After = after
	return result, nil
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func runPythonDancerForVideo(videoPath, outputPath, pythonDancerDir string, py pythonExecSpec, postProcessMode string) (bool, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()
	postProcessMode = normalizePostProcessMode(postProcessMode)

	runMode := func(convert bool, profile pythonDancerTuningProfile) (bool, string, error) {
		workDir, err := os.MkdirTemp("", "python-dancer-*")
		if err != nil {
			return false, "", err
		}
		defer os.RemoveAll(workDir)

		args := append([]string{}, py.ArgsPrefix...)
		args = append(args, buildPythonDancerCLIArgs(videoPath, outputPath, convert, profile)...)
		cmd := buildCmd(py.Name, args...)
		cmd.Dir = workDir
		cmd.Env = commandEnvWithPythonDancer(pythonDancerDir)

		output, err := runPythonDancerCommand(ctx, cmd)
		if err == nil && fileExistsNonEmpty(outputPath) {
			return convert, output, nil
		}
		if err != nil && fileExistsNonEmpty(outputPath) && isIgnorablePythonDancerError(output) {
			return convert, appendOutputNote(output, "heatmap generation failed but funscript was created"), nil
		}
		if err == nil && !fileExistsNonEmpty(outputPath) {
			err = fmt.Errorf("PythonDancer finished without creating %s", outputPath)
		}
		return convert, output, err
	}

	profiles := defaultPythonDancerProfiles()
	allOutput := make([]string, 0, len(profiles))
	for idx, profile := range profiles {
		usedConvert, output, err := runMode(false, profile)
		if err != nil {
			removeGeneratedArtifacts(outputPath)
			usedConvert, convertOutput, convertErr := runMode(true, profile)
			output = strings.TrimSpace(strings.Join([]string{output, convertOutput}, "\n"))
			err = convertErr
			if err != nil {
				allOutput = append(allOutput, appendOutputNote(output, fmt.Sprintf("profile=%s convert=%t", profile.Name, usedConvert)))
				continue
			}
		}

		report, qualityErr := analyzeGeneratedFunscript(outputPath)
		qualityLine := ""
		if qualityErr != nil {
			qualityLine = fmt.Sprintf("quality_check_error=%v", qualityErr)
		} else {
			qualityLine = formatQualityReport(report)
		}
		postLine := ""
		if qualityErr == nil {
			postResult, postErr := postProcessGeneratedFunscript(outputPath, report, postProcessMode)
			if postErr != nil {
				postLine = fmt.Sprintf("post_process_error=%v", postErr)
			} else if postResult.Applied {
				report = postResult.After
				qualityLine = formatQualityReport(report)
				postLine = fmt.Sprintf("post_process applied before=%s after=%s",
					formatQualityReport(postResult.Before),
					formatQualityReport(postResult.After),
				)
			}
		}
		attemptSummary := fmt.Sprintf("profile=%s convert=%t %s", profile.Name, usedConvert, qualityLine)
		if postLine != "" {
			attemptSummary = attemptSummary + " " + postLine
		}
		output = appendOutputNote(output, attemptSummary)
		allOutput = append(allOutput, output)

		if qualityErr == nil && postProcessMode == postProcessModeAuto && qualityNeedsRetry(report) && idx < len(profiles)-1 {
			removeGeneratedArtifacts(outputPath)
			allOutput = append(allOutput, fmt.Sprintf("retrying with gentler PythonDancer profile because %s", qualityLine))
			continue
		}
		return usedConvert, strings.TrimSpace(strings.Join(allOutput, "\n")), nil
	}

	combinedOutput := strings.TrimSpace(strings.Join(allOutput, "\n"))
	if combinedOutput != "" {
		return false, combinedOutput, fmt.Errorf("PythonDancer did not produce an acceptable funscript")
	}
	return false, "", fmt.Errorf("PythonDancer did not produce an acceptable funscript")
}

func escapedFilenameLike(s string) string {
	var buffer bytes.Buffer
	json.HTMLEscape(&buffer, []byte(s))
	return buffer.String()
}

func appendSceneFilename(scene *models.Scene, filename string) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return
	}

	var filenames []string
	_ = json.Unmarshal([]byte(scene.FilenamesArr), &filenames)
	for _, existing := range filenames {
		if existing == filename {
			return
		}
	}
	filenames = append(filenames, filename)
	if data, err := json.Marshal(filenames); err == nil {
		scene.FilenamesArr = string(data)
	}
}

func linkGeneratedScriptToScene(db *gorm.DB, video *models.File, script *models.File) (string, error) {
	if video.SceneID != 0 {
		script.SceneID = video.SceneID
		if err := script.Save(); err != nil {
			return "", err
		}
		var scene models.Scene
		if err := scene.GetIfExistByPK(video.SceneID); err == nil {
			appendSceneFilename(&scene, video.Filename)
			appendSceneFilename(&scene, script.Filename)
			if err := scene.Save(); err != nil {
				return "", err
			}
			scene.UpdateStatus()
			return scene.SceneID, nil
		}
		return "", nil
	}

	filename := escapedFilenameLike(video.Filename)
	var scenes []models.Scene
	if err := db.Where("filenames_arr LIKE ?", `%"`+filename+`"%`).Find(&scenes).Error; err != nil {
		return "", err
	}
	if len(scenes) != 1 {
		return "", nil
	}

	video.SceneID = scenes[0].ID
	if err := video.Save(); err != nil {
		return "", err
	}
	script.SceneID = scenes[0].ID
	if err := script.Save(); err != nil {
		return "", err
	}

	appendSceneFilename(&scenes[0], video.Filename)
	appendSceneFilename(&scenes[0], script.Filename)
	if err := scenes[0].Save(); err != nil {
		return "", err
	}
	scenes[0].UpdateStatus()
	return scenes[0].SceneID, nil
}

func scanGeneratedScriptFile(path string, volID uint) (models.File, error) {
	db, _ := models.GetDB()
	defer db.Close()

	var fl models.File
	db.Where(&models.File{
		Path:     filepath.Dir(path),
		Filename: filepath.Base(path),
		Type:     "script",
	}).FirstOrCreate(&fl)

	fStat, err := os.Stat(path)
	if err != nil {
		return fl, err
	}
	fTimes, err := times.Stat(path)
	if err != nil {
		return fl, err
	}

	fl.Size = fStat.Size()
	fl.CreatedTime = fTimes.ModTime()
	fl.UpdatedTime = fTimes.ModTime()
	fl.VolumeID = volID
	fl.HasHeatmap = false
	if duration, err := getFunscriptDuration(path); err == nil {
		fl.VideoDuration = duration
	}
	if err := fl.Save(); err != nil {
		return fl, err
	}
	return fl, nil
}

func renderGeneratedHeatmap(scriptPath string, scriptFile *models.File) error {
	if scriptFile == nil || scriptFile.ID == 0 {
		return fmt.Errorf("script file is not persisted")
	}

	if err := os.MkdirAll(common.ScriptHeatmapDir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(common.ScriptHeatmapDir, fmt.Sprintf("heatmap-%d.png", scriptFile.ID))
	if err := RenderHeatmap(scriptPath, dest, 1000, 10, 250); err != nil {
		return err
	}

	scriptFile.HasHeatmap = true
	scriptFile.RefreshHeatmapCache = true
	return scriptFile.Save()
}

func GeneratePythonDancerFunscripts(req PythonDancerBatchRequest) (*PythonDancerBatchResult, int, error) {
	limit := normalizePythonDancerBatchLimit(req.Limit)
	concurrency := normalizePythonDancerBatchConcurrency(req.Concurrency)
	postProcessMode := normalizePostProcessMode(req.PostProcessMode)

	pythonDancerDir, err := findPythonDancerDir()
	if err != nil {
		return nil, 500, err
	}
	pyExec, err := findPythonExec(pythonDancerDir)
	if err != nil {
		return nil, 500, err
	}

	db, _ := models.GetDB()
	defer db.Close()

	query := db.Model(&models.File{}).
		Joins("left join volumes on files.volume_id = volumes.id").
		Joins("left join scenes on files.scene_id = scenes.id").
		Where("files.type = ?", "video").
		Where("volumes.type = ?", "local")
	if req.FileID != 0 {
		query = query.Where("files.id = ?", req.FileID)
	}
	if req.VolumeID != 0 {
		query = query.Where("files.volume_id = ?", req.VolumeID)
	}
	if strings.TrimSpace(req.PathPrefix) != "" {
		query = query.Where("files.path LIKE ?", strings.TrimSpace(req.PathPrefix)+"%")
	}

	var files []models.File
	orderExpr := "COALESCE(NULLIF(scenes.release_date, '0001-01-01 00:00:00'), NULLIF(scenes.release_date, '0001-01-01T00:00:00Z'), scenes.added_date, files.created_time) DESC"
	query = query.Order(orderExpr)
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&files).Error; err != nil {
		return nil, 500, err
	}
	if req.FileID != 0 && len(files) == 0 {
		return nil, 404, fmt.Errorf("video file %d not found", req.FileID)
	}

	out := &PythonDancerBatchResult{
		Scanned: len(files),
		Results: make([]PythonDancerBatchItem, len(files)),
	}
	if len(files) == 0 {
		return out, 200, nil
	}
	if concurrency > len(files) {
		concurrency = len(files)
	}

	type batchJob struct {
		Index int
		File  models.File
	}
	type batchResult struct {
		Index int
		Item  PythonDancerBatchItem
	}

	jobs := make(chan batchJob)
	results := make(chan batchResult, len(files))

	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for job := range jobs {
			video := job.File
			item := PythonDancerBatchItem{
				FileID:   video.ID,
				Filename: video.Filename,
			}

			videoPath := video.GetPath()
			outputPath := siblingFunscriptPath(videoPath)
			item.OutputPath = outputPath
			_ = os.Remove(siblingHeatmapPath(outputPath))

			if req.ForceRegenerate {
				removeGeneratedArtifacts(outputPath)
			}

			if fileExistsNonEmpty(outputPath) {
				item.Message = "skipped: funscript already exists"
				results <- batchResult{Index: job.Index, Item: item}
				continue
			}

			usedConvert, cmdOutput, err := runPythonDancerForVideo(videoPath, outputPath, pythonDancerDir, pyExec, postProcessMode)
			item.UsedConvert = usedConvert
			if err != nil {
				item.Error = err.Error()
				if strings.TrimSpace(cmdOutput) != "" {
					item.Message = cmdOutput
				}
				results <- batchResult{Index: job.Index, Item: item}
				continue
			}

			scriptFile, err := scanGeneratedScriptFile(outputPath, video.VolumeID)
			if err != nil {
				item.Error = err.Error()
				results <- batchResult{Index: job.Index, Item: item}
				continue
			}
			if err := renderGeneratedHeatmap(outputPath, &scriptFile); err != nil {
				item.Error = err.Error()
				results <- batchResult{Index: job.Index, Item: item}
				continue
			}
			_ = os.Remove(siblingHeatmapPath(outputPath))

			db, _ := models.GetDB()
			sceneID, matchErr := linkGeneratedScriptToScene(db, &video, &scriptFile)
			db.Close()
			if matchErr != nil {
				item.Error = matchErr.Error()
				results <- batchResult{Index: job.Index, Item: item}
				continue
			}

			if strings.TrimSpace(cmdOutput) != "" {
				item.Message = cmdOutput
			} else if usedConvert {
				item.Message = "generated via PythonDancer with convert retry"
			} else {
				item.Message = "generated via PythonDancer"
			}
			item.SceneID = sceneID
			results <- batchResult{Index: job.Index, Item: item}
		}
	}

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}

	go func() {
		for i, file := range files {
			jobs <- batchJob{Index: i, File: file}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	for result := range results {
		out.Results[result.Index] = result.Item
	}

	for _, item := range out.Results {
		if item.Error != "" {
			out.Errors++
			continue
		}
		if strings.Contains(item.Message, "skipped: funscript already exists") {
			out.SkippedExisting++
			continue
		}
		out.Generated++
		if item.SceneID != "" {
			out.Matched++
		}
	}

	return out, 200, nil
}

func RunPythonDancerFunscriptTask(req PythonDancerBatchRequest) {
	tlog := log.WithField("task", "python-dancer-funscripts")
	if models.CheckLock("python-dancer-funscripts") {
		tlog.Infof("skipped: task already running")
		return
	}

	models.CreateLock("python-dancer-funscripts")
	defer models.RemoveLock("python-dancer-funscripts")

	limitLabel := "all"
	if normalizedLimit := normalizePythonDancerBatchLimit(req.Limit); normalizedLimit > 0 {
		limitLabel = strconv.Itoa(normalizedLimit)
	}
	tlog.Infof("start limit=%s concurrency=%d volume_id=%d path_prefix=%q",
		limitLabel, normalizePythonDancerBatchConcurrency(req.Concurrency), req.VolumeID, req.PathPrefix)
	result, statusCode, err := GeneratePythonDancerFunscripts(req)
	if err != nil {
		tlog.Errorf("failed status=%d err=%v", statusCode, err)
		return
	}
	tlog.Infof("done status=%d scanned=%d generated=%d matched=%d skipped_existing=%d errors=%d",
		statusCode, result.Scanned, result.Generated, result.Matched, result.SkippedExisting, result.Errors)
}
