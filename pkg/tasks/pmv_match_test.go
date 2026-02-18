package tasks

import (
	"testing"

	"github.com/xbapps/xbvr/pkg/common"
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

func TestExtractJSONObject(t *testing.T) {
	raw := "```json\n{\"best_index\":1,\"best_confidence\":0.9}\n```"
	got := extractJSONObject(raw)
	if got != "{\"best_index\":1,\"best_confidence\":0.9}" {
		t.Fatalf("unexpected extracted json: %q", got)
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

func TestOpenAIPMVModelFallback(t *testing.T) {
	orig := common.EnvConfig.OpenAIPMVModel
	t.Cleanup(func() { common.EnvConfig.OpenAIPMVModel = orig })

	common.EnvConfig.OpenAIPMVModel = ""
	if got := openAIPMVModel(); got != defaultOpenAIModel {
		t.Fatalf("expected fallback model %q, got %q", defaultOpenAIModel, got)
	}

	common.EnvConfig.OpenAIPMVModel = "gpt-4.1-mini"
	if got := openAIPMVModel(); got != "gpt-4.1-mini" {
		t.Fatalf("expected configured model, got %q", got)
	}
}
