package tasks

import (
	"testing"

	"github.com/xbapps/xbvr/pkg/scrape"
)

func TestNormalizePMVQuery(t *testing.T) {
	in := "My.Cool-Video_6k_60fps_vr_SBS.mp4"
	got := normalizePMVQuery(in)
	want := "my cool video"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestScorePMVCandidatesByText(t *testing.T) {
	query := "amazing sunset remix"
	candidates := []scrape.PMVHavenCandidate{
		{ID: "a", Title: "Random Compilation", SceneURL: "https://pmvhaven.com/random"},
		{ID: "b", Title: "Amazing Sunset Remix", SceneURL: "https://pmvhaven.com/amazing-sunset-remix"},
	}

	ranked := scorePMVCandidatesByText(query, candidates)
	if len(ranked) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ranked))
	}
	if ranked[0].PMVID != "b" {
		t.Fatalf("expected best candidate id b, got %s", ranked[0].PMVID)
	}
	if ranked[0].Confidence <= ranked[1].Confidence {
		t.Fatalf("expected candidate b to have higher confidence")
	}
}

func TestNormalizePMVQuery_StripsChannelAndNoiseTokens(t *testing.T) {
	in := "DigitalFiend_-_Super_Smash_Hoes_1771348033231_xl73j501.mp4"
	got := normalizePMVQuery(in)
	want := "super smash hoes"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizePMVQuery_ChannelDelimiterTitleOnly(t *testing.T) {
	in := "wezzam_-_are_u_gooning_again.mp4"
	got := normalizePMVQuery(in)
	want := "are u gooning again"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizePMVQuery_StripsTimestampRandomSuffix(t *testing.T) {
	in := "CrimsonPMV_-_MultiStroke_-_Gooner_PMV_Crimson_PMV_1768217534327_eokwfllu.mp4"
	got := normalizePMVQuery(in)
	want := "multi stroke gooner pmv crimson pmv"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizePMVQuery_SplitsCamelCase(t *testing.T) {
	in := "Instagram Models - Try Not To Cum (TheChillPanda).mp4"
	got := normalizePMVQuery(in)
	want := "try not to cum the chill panda"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildPMVSearchQueries_DropsLeadingChannelToken(t *testing.T) {
	filename := "YourVideosCouldLookBetter_-_ADHDPMV_-_4k60_-_Feminist_to_Daddy's_Girl_-_World_PMV_Games_2025_upscale.mp4"
	base := normalizePMVQuery(filename)
	if base != "adhdpmv feminist to daddy's girl world pmv games" {
		t.Fatalf("unexpected base query %q", base)
	}
	queries := buildPMVSearchQueries(filename, base)

	found := false
	for _, q := range queries {
		if q == "feminist to daddy's girl world pmv games" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected fallback query without channel/noise token, got %v", queries)
	}
}

func TestInferPMVStudio_FromCandidateTitlePrefix(t *testing.T) {
	filename := "YourVideosCouldLookBetter_-_ADHDPMV_-_4k60_-_Feminist_to_Daddy's_Girl_-_World_PMV_Games_2025_upscale.mp4"
	title := "ADHDPMV - 4k60 - Feminist to Daddy's Girl - World PMV Games 2025 upscale"
	if got := inferPMVStudio(filename, title); got != "ADHDPMV" {
		t.Fatalf("expected ADHDPMV, got %q", got)
	}
}

func TestInferPMVStudio_FromFilenameFallback(t *testing.T) {
	filename := "CrimsonPMV_-_MultiStroke_-_Gooner_PMV_Crimson_PMV_1768217534327_eokwfllu.mp4"
	if got := inferPMVStudio(filename, ""); got != "CrimsonPMV" {
		t.Fatalf("expected CrimsonPMV, got %q", got)
	}
}

func TestNormalizePMVBatchLimit(t *testing.T) {
	if got := normalizePMVBatchLimit(0); got != 50 {
		t.Fatalf("expected default limit 50, got %d", got)
	}
	if got := normalizePMVBatchLimit(-10); got != 50 {
		t.Fatalf("expected default limit 50, got %d", got)
	}
	if got := normalizePMVBatchLimit(999); got != 500 {
		t.Fatalf("expected max limit 500, got %d", got)
	}
	if got := normalizePMVBatchLimit(123); got != 123 {
		t.Fatalf("expected passthrough limit, got %d", got)
	}
}

func TestNormalizePMVBatchConcurrency(t *testing.T) {
	if got := normalizePMVBatchConcurrency(0); got != 10 {
		t.Fatalf("expected default concurrency 10, got %d", got)
	}
	if got := normalizePMVBatchConcurrency(-5); got != 10 {
		t.Fatalf("expected default concurrency 10, got %d", got)
	}
	if got := normalizePMVBatchConcurrency(99); got != 50 {
		t.Fatalf("expected max concurrency 50, got %d", got)
	}
	if got := normalizePMVBatchConcurrency(12); got != 12 {
		t.Fatalf("expected passthrough concurrency, got %d", got)
	}
}
