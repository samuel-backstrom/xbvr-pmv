package tasks

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/jinzhu/gorm"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
)

const (
	pmvMatchCandidateLimit = 5
	pmvAutoLinkThreshold   = 0.85
	defaultOpenAIModel     = "gpt-5"
)

type PMVMatchCandidate struct {
	Rank         int     `json:"rank"`
	PMVID        string  `json:"pmv_id"`
	Title        string  `json:"title"`
	SceneURL     string  `json:"scene_url"`
	ThumbnailURL string  `json:"thumbnail_url"`
	Confidence   float64 `json:"confidence"`
	Reason       string  `json:"reason"`
}

type PMVMatchResult struct {
	Status         string              `json:"status"`
	FileID         uint                `json:"file_id"`
	Query          string              `json:"query"`
	Autolinked     bool                `json:"autolinked"`
	Confidence     float64             `json:"confidence"`
	MatchedSceneID string              `json:"matched_scene_id,omitempty"`
	Candidates     []PMVMatchCandidate `json:"candidates"`
	Message        string              `json:"message,omitempty"`
}

type openAIPMVRankResult struct {
	BestIndex      int                      `json:"best_index"`
	BestConfidence float64                  `json:"best_confidence"`
	Candidates     []openAIPMVRankCandidate `json:"candidates"`
}

type openAIPMVRankCandidate struct {
	Index      int     `json:"index"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

func MatchPMVFile(fileID uint, dryRun bool) (*PMVMatchResult, int, error) {
	if fileID == 0 {
		return nil, 400, errors.New("file_id is required")
	}

	db, _ := models.GetDB()
	defer db.Close()

	var file models.File
	err := db.Where(&models.File{ID: fileID}).First(&file).Error
	if err == gorm.ErrRecordNotFound {
		return nil, 404, fmt.Errorf("file_id %d was not found", fileID)
	}
	if err != nil {
		return nil, 500, err
	}
	if file.SceneID != 0 {
		return nil, 409, fmt.Errorf("file_id %d is already matched", fileID)
	}

	query := normalizePMVQuery(file.Filename)
	if query == "" {
		return nil, 400, errors.New("could not build a query from filename")
	}
	tlog := log.WithField("task", "pmv-match").WithField("file_id", fileID)
	tlog.Infof("start filename=%q query=%q dry_run=%v", file.Filename, query, dryRun)

	result := &PMVMatchResult{
		Status:     "ok",
		FileID:     fileID,
		Query:      query,
		Candidates: []PMVMatchCandidate{},
	}

	candidates, err := scrape.SearchPMVHaven(query, pmvMatchCandidateLimit)
	if err != nil {
		tlog.Errorf("search failed query=%q err=%v", query, err)
		return nil, 424, err
	}
	if len(candidates) == 0 {
		tlog.Infof("search returned 0 candidates")
		result.Message = "no PMVHaven candidates found"
		return result, 200, nil
	}
	for i, c := range candidates {
		tlog.Infof("parsed candidate #%d title=%q scene_url=%q thumbnail_url=%q", i+1, c.Title, c.SceneURL, c.ThumbnailURL)
	}

	ranked := scorePMVCandidatesByText(query, candidates)
	if len(ranked) > 0 {
		tlog.Infof("baseline top title=%q pmv_id=%s confidence=%.3f", ranked[0].Title, ranked[0].PMVID, ranked[0].Confidence)
	}

	openAIRankMsg := ""
	if common.EnvConfig.OpenAIAPIKey == "" {
		openAIRankMsg = "OPENAI_API_KEY missing, used baseline text ranking only"
		tlog.Infof("openai ranking skipped missing OPENAI_API_KEY")
	} else {
		model := openAIPMVModel()
		tlog.Infof("openai ranking requested model=%s candidates=%d", model, len(ranked))
		openAIRanks, err := rankPMVCandidatesWithOpenAI(query, ranked)
		if err != nil {
			openAIRankMsg = fmt.Sprintf("OpenAI ranking unavailable on model %s (%v), used baseline text ranking only", model, err)
			tlog.Warnf("openai ranking unavailable model=%s err=%v", model, err)
		} else {
			mergeOpenAIRanks(ranked, openAIRanks)
			openAIRankMsg = fmt.Sprintf("OpenAI ranking applied with model %s", model)
			tlog.Infof("openai ranking applied model=%s", model)
		}
	}
	sortCandidates(ranked)

	for i := range ranked {
		ranked[i].Rank = i + 1
	}
	result.Candidates = ranked
	if len(ranked) > 0 {
		result.Confidence = ranked[0].Confidence
		result.MatchedSceneID = "pmvhaven-" + ranked[0].PMVID
		tlog.Infof("final top parsed title=%q pmv_id=%s confidence=%.3f thumbnail=%q", ranked[0].Title, ranked[0].PMVID, ranked[0].Confidence, ranked[0].ThumbnailURL)
	}

	if len(ranked) == 0 || ranked[0].Confidence < pmvAutoLinkThreshold {
		result.Message = fmt.Sprintf("best confidence %.2f is below autolink threshold %.2f", result.Confidence, pmvAutoLinkThreshold)
		if openAIRankMsg != "" {
			result.Message = result.Message + "; " + openAIRankMsg
		}
		tlog.Infof("not autolinked best_confidence=%.3f threshold=%.2f", result.Confidence, pmvAutoLinkThreshold)
		return result, 200, nil
	}

	if dryRun {
		result.Message = "dry run: best candidate found, no database changes applied"
		if openAIRankMsg != "" {
			result.Message = result.Message + "; " + openAIRankMsg
		}
		tlog.Infof("dry run success candidate_title=%q pmv_id=%s confidence=%.3f", ranked[0].Title, ranked[0].PMVID, ranked[0].Confidence)
		return result, 200, nil
	}

	matchedSceneID, err := applyPMVMatch(db, &file, ranked[0])
	if err != nil {
		tlog.Errorf("apply match failed candidate_title=%q pmv_id=%s err=%v", ranked[0].Title, ranked[0].PMVID, err)
		return nil, 500, err
	}

	result.Autolinked = true
	result.MatchedSceneID = matchedSceneID
	result.Message = "file linked to PMVHaven scene"
	if openAIRankMsg != "" {
		result.Message = result.Message + "; " + openAIRankMsg
	}
	tlog.Infof("autolinked scene_id=%s candidate_title=%q confidence=%.3f", matchedSceneID, ranked[0].Title, ranked[0].Confidence)
	return result, 200, nil
}

func normalizePMVQuery(filename string) string {
	name := strings.TrimSpace(filename)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.ToLower(name)

	// Common PMV filename format: channel_-_title.
	if idx := strings.Index(name, "_-_"); idx >= 0 {
		right := strings.TrimSpace(name[idx+3:])
		if right != "" {
			name = right
		}
	}

	// Some exporters use "channel - title"; only strip when lhs looks like a short channel token.
	for _, sep := range []string{" - ", " – ", " — "} {
		if idx := strings.Index(name, sep); idx > 0 {
			left := strings.TrimSpace(name[:idx])
			right := strings.TrimSpace(name[idx+len(sep):])
			if right != "" && len(strings.Fields(left)) <= 2 {
				name = right
				break
			}
		}
	}

	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, ".", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "(", " ")
	name = strings.ReplaceAll(name, ")", " ")
	name = regexp.MustCompile(`[^a-z0-9\s]+`).ReplaceAllString(name, " ")
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	skipTokens := map[string]bool{
		"1080p": true, "1440p": true, "2160p": true, "4k": true, "5k": true, "6k": true, "8k": true,
		"fps": true, "60fps": true, "30fps": true,
		"sbs": true, "tb": true, "lr": true, "vr": true,
		"mp4": true, "mkv": true, "avi": true, "mov": true, "wmv": true,
		"uhd": true, "hd": true, "fullhd": true, "hq": true,
	}

	out := make([]string, 0)
	for _, tok := range strings.Fields(name) {
		if skipTokens[tok] {
			continue
		}
		if isLikelyNoiseToken(tok) {
			continue
		}
		out = append(out, tok)
	}
	if len(out) == 0 {
		return strings.Join(strings.Fields(name), " ")
	}
	return strings.Join(out, " ")
}

func isLikelyNoiseToken(tok string) bool {
	tok = strings.TrimSpace(strings.ToLower(tok))
	if tok == "" {
		return false
	}

	digits := 0
	letters := 0
	vowels := 0
	for _, r := range tok {
		if r >= '0' && r <= '9' {
			digits++
			continue
		}
		if r >= 'a' && r <= 'z' {
			letters++
			switch r {
			case 'a', 'e', 'i', 'o', 'u':
				vowels++
			}
		}
	}

	// Long numeric IDs are never useful for PMV title search.
	if digits == len(tok) && len(tok) >= 4 {
		return true
	}

	// Mixed long alnum tokens are typically upload IDs / random suffixes.
	if len(tok) >= 10 && digits > 0 && letters > 0 {
		return true
	}

	// Shorter mixed tokens can still be random IDs like xl73j501.
	if len(tok) >= 8 && digits >= 3 && letters >= 3 {
		if vowels == 0 {
			return true
		}
		if float64(digits)/float64(len(tok)) >= 0.35 {
			return true
		}
	}

	return false
}

func scorePMVCandidatesByText(query string, candidates []scrape.PMVHavenCandidate) []PMVMatchCandidate {
	queryTokens := tokenSet(query)
	out := make([]PMVMatchCandidate, 0, len(candidates))
	for _, c := range candidates {
		titleTokens := tokenSet(c.Title)
		overlap := overlapScore(queryTokens, titleTokens)

		titleLower := strings.ToLower(c.Title)
		queryLower := strings.ToLower(query)
		containsBonus := 0.0
		if queryLower != "" && strings.Contains(titleLower, queryLower) {
			containsBonus = 0.2
		}

		confidence := clampScore(0.15 + overlap*0.7 + containsBonus)
		out = append(out, PMVMatchCandidate{
			PMVID:        c.ID,
			Title:        c.Title,
			SceneURL:     c.SceneURL,
			ThumbnailURL: c.ThumbnailURL,
			Confidence:   confidence,
			Reason:       "baseline text similarity",
		})
	}
	sortCandidates(out)
	return out
}

func rankPMVCandidatesWithOpenAI(query string, candidates []PMVMatchCandidate) ([]openAIPMVRankCandidate, error) {
	type openAIMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type openAIRequest struct {
		Model       string          `json:"model"`
		Temperature float64         `json:"temperature"`
		Messages    []openAIMessage `json:"messages"`
	}

	prompt := strings.Builder{}
	prompt.WriteString("Match the best PMV candidate for this local filename query.\n")
	prompt.WriteString("Return JSON only with this schema:\n")
	prompt.WriteString(`{"best_index":1,"best_confidence":0.0,"candidates":[{"index":1,"confidence":0.0,"reason":"short reason"}]}` + "\n")
	prompt.WriteString("Confidence must be between 0.0 and 1.0.\n")
	prompt.WriteString(fmt.Sprintf("Query: %q\n", query))
	prompt.WriteString("Candidates:\n")
	for i, c := range candidates {
		prompt.WriteString(fmt.Sprintf("%d) title=%q scene_url=%q thumbnail_url=%q\n", i+1, c.Title, c.SceneURL, c.ThumbnailURL))
	}

	reqBody := openAIRequest{
		Model:       openAIPMVModel(),
		Temperature: 0,
		Messages: []openAIMessage{
			{
				Role:    "system",
				Content: "You are a strict JSON responder for filename-to-title matching.",
			},
			{
				Role:    "user",
				Content: prompt.String(),
			},
		},
	}

	resp, err := resty.New().R().
		SetHeader("Authorization", "Bearer "+common.EnvConfig.OpenAIAPIKey).
		SetHeader("Content-Type", "application/json").
		SetBody(reqBody).
		Post("https://api.openai.com/v1/chat/completions")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		msg := strings.TrimSpace(gjson.Get(resp.String(), "error.message").String())
		if msg != "" {
			return nil, fmt.Errorf("openai request failed with status %d (%s)", resp.StatusCode(), msg)
		}
		return nil, fmt.Errorf("openai request failed with status %d", resp.StatusCode())
	}

	content := gjson.Get(resp.String(), "choices.0.message.content").String()
	if strings.TrimSpace(content) == "" {
		return nil, errors.New("openai returned empty response")
	}
	jsonContent := extractJSONObject(content)
	if jsonContent == "" {
		return nil, errors.New("openai response did not contain JSON")
	}

	var parsed openAIPMVRankResult
	if err := json.Unmarshal([]byte(jsonContent), &parsed); err != nil {
		return nil, err
	}
	return parsed.Candidates, nil
}

func openAIPMVModel() string {
	model := strings.TrimSpace(common.EnvConfig.OpenAIPMVModel)
	if model == "" {
		return defaultOpenAIModel
	}
	return model
}

func mergeOpenAIRanks(candidates []PMVMatchCandidate, openAIRanks []openAIPMVRankCandidate) {
	if len(candidates) == 0 || len(openAIRanks) == 0 {
		return
	}
	for _, rank := range openAIRanks {
		idx := rank.Index - 1
		if idx < 0 || idx >= len(candidates) {
			continue
		}
		conf := clampScore(rank.Confidence)
		if conf == 0 {
			continue
		}
		candidates[idx].Confidence = conf
		if strings.TrimSpace(rank.Reason) != "" {
			candidates[idx].Reason = strings.TrimSpace(rank.Reason)
		}
	}
}

func applyPMVMatch(db *gorm.DB, file *models.File, candidate PMVMatchCandidate) (string, error) {
	sceneSuffix := strings.TrimSpace(candidate.PMVID)
	if sceneSuffix == "" {
		sceneSuffix = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(candidate.Title), "-")
		sceneSuffix = strings.Trim(sceneSuffix, "-")
		if sceneSuffix == "" {
			sceneSuffix = "matched"
		}
	}
	sceneID := "pmvhaven-" + sceneSuffix

	ext := models.ScrapedScene{
		SceneID:     sceneID,
		ScraperID:   "pmvhaven",
		SceneType:   "VR",
		Title:       strings.TrimSpace(candidate.Title),
		Studio:      "PMVHaven",
		Site:        "PMVHaven",
		HomepageURL: strings.TrimSpace(candidate.SceneURL),
		MembersUrl:  strings.TrimSpace(candidate.SceneURL),
		Filenames:   []string{file.Filename},
	}
	if strings.TrimSpace(candidate.ThumbnailURL) != "" {
		ext.Covers = append(ext.Covers, strings.TrimSpace(candidate.ThumbnailURL))
	}

	if err := models.SceneCreateUpdateFromExternal(db, ext); err != nil {
		return "", err
	}

	var scene models.Scene
	if err := scene.GetIfExist(sceneID); err != nil {
		return "", err
	}

	file.SceneID = scene.ID
	if err := file.Save(); err != nil {
		return "", err
	}

	var filenames []string
	_ = json.Unmarshal([]byte(scene.FilenamesArr), &filenames)
	exists := false
	for _, fn := range filenames {
		if fn == file.Filename {
			exists = true
			break
		}
	}
	if !exists {
		filenames = append(filenames, file.Filename)
		if b, err := json.Marshal(filenames); err == nil {
			scene.FilenamesArr = string(b)
		}
	}

	models.AddAction(scene.SceneID, "match", "filenames_arr", scene.FilenamesArr)
	scene.UpdateStatus()

	IndexScenes(&[]models.Scene{scene})
	return scene.SceneID, nil
}

func tokenSet(s string) map[string]bool {
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9\s]+`).ReplaceAllString(s, " ")
	out := map[string]bool{}
	for _, tok := range strings.Fields(s) {
		if len(tok) < 2 {
			continue
		}
		out[tok] = true
	}
	return out
}

func overlapScore(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersections := 0
	for k := range a {
		if b[k] {
			intersections++
		}
	}
	return float64(intersections) / float64(len(a))
}

func clampScore(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func sortCandidates(c []PMVMatchCandidate) {
	sort.Slice(c, func(i, j int) bool {
		if c[i].Confidence == c[j].Confidence {
			return c[i].Title < c[j].Title
		}
		return c[i].Confidence > c[j].Confidence
	})
}

func extractJSONObject(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	// Strip markdown fences if present.
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(text[start : end+1])
}
