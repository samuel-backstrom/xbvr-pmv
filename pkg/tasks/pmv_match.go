package tasks

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
)

const (
	pmvMatchCandidateLimit = 5
)

type PMVMatchCandidate struct {
	Rank         int     `json:"rank"`
	PMVID        string  `json:"pmv_id"`
	Title        string  `json:"title"`
	SceneURL     string  `json:"scene_url"`
	ThumbnailURL string  `json:"thumbnail_url"`
	Channel      string  `json:"channel,omitempty"`
	Description  string  `json:"description,omitempty"`
	Confidence   float64 `json:"-"`
	Reason       string  `json:"reason"`
}

type PMVMatchResult struct {
	Status         string              `json:"status"`
	FileID         uint                `json:"file_id"`
	Filename       string              `json:"filename"`
	Query          string              `json:"query"`
	Autolinked     bool                `json:"autolinked"`
	MatchedSceneID string              `json:"matched_scene_id,omitempty"`
	Candidates     []PMVMatchCandidate `json:"candidates"`
	Message        string              `json:"message,omitempty"`
}

type PMVMatchBatchRequest struct {
	DryRun            bool   `json:"dry_run"`
	Limit             int    `json:"limit"`
	Concurrency       int    `json:"concurrency,omitempty"`
	VolumeID          uint   `json:"volume_id,omitempty"`
	PathPrefix        string `json:"path_prefix,omitempty"`
	RefreshExisting   bool   `json:"refresh_existing,omitempty"`
	UpdateTitle       bool   `json:"update_title,omitempty"`
	UpdateStudio      bool   `json:"update_studio,omitempty"`
	UpdateSceneURL    bool   `json:"update_scene_url,omitempty"`
	UpdateThumbnail   bool   `json:"update_thumbnail,omitempty"`
	UpdateDescription bool   `json:"update_description,omitempty"`
}

type PMVMatchBatchItem struct {
	FileID     uint            `json:"file_id"`
	SceneID    string          `json:"scene_id,omitempty"`
	Filename   string          `json:"filename"`
	StatusCode int             `json:"status_code"`
	Updated    bool            `json:"updated,omitempty"`
	Message    string          `json:"message,omitempty"`
	Error      string          `json:"error,omitempty"`
	Result     *PMVMatchResult `json:"result,omitempty"`
}

type PMVMatchBatchResult struct {
	Scanned             int                 `json:"scanned"`
	Matched             int                 `json:"matched"`
	SkippedAlreadyMatch int                 `json:"skipped_already_matched"`
	Errors              int                 `json:"errors"`
	Updated             int                 `json:"updated,omitempty"`
	SkippedNoPMVURL     int                 `json:"skipped_no_pmv_url,omitempty"`
	Mode                string              `json:"mode,omitempty"`
	Results             []PMVMatchBatchItem `json:"results"`
}

type pmvMetadataUpdateOptions struct {
	Title       bool
	Studio      bool
	SceneURL    bool
	Thumbnail   bool
	Description bool
}

func normalizePMVMetadataUpdateOptions(req PMVMatchBatchRequest) pmvMetadataUpdateOptions {
	opts := pmvMetadataUpdateOptions{
		Title:       req.UpdateTitle,
		Studio:      req.UpdateStudio,
		SceneURL:    req.UpdateSceneURL,
		Thumbnail:   req.UpdateThumbnail,
		Description: req.UpdateDescription,
	}
	if !opts.Title && !opts.Studio && !opts.SceneURL && !opts.Thumbnail && !opts.Description {
		// Safe default for refresh mode: update all PMVHaven-backed metadata.
		opts = pmvMetadataUpdateOptions{
			Title:       true,
			Studio:      true,
			SceneURL:    true,
			Thumbnail:   true,
			Description: true,
		}
	}
	return opts
}

func (o pmvMetadataUpdateOptions) String() string {
	keys := make([]string, 0, 5)
	if o.Title {
		keys = append(keys, "title")
	}
	if o.Studio {
		keys = append(keys, "studio")
	}
	if o.SceneURL {
		keys = append(keys, "scene_url")
	}
	if o.Thumbnail {
		keys = append(keys, "thumbnail")
	}
	if o.Description {
		keys = append(keys, "description")
	}
	return strings.Join(keys, ",")
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
		Filename:   file.Filename,
		Query:      query,
		Candidates: []PMVMatchCandidate{},
	}

	searchQueries := buildPMVSearchQueries(file.Filename, query)
	var candidates []scrape.PMVHavenCandidate
	var searchErr error
	usedQuery := query
	for i, q := range searchQueries {
		tlog.Infof("search attempt=%d/%d query=%q", i+1, len(searchQueries), q)
		candidates, searchErr = scrape.SearchPMVHaven(q, pmvMatchCandidateLimit)
		if searchErr != nil {
			tlog.Warnf("search failed query=%q err=%v", q, searchErr)
			continue
		}
		if len(candidates) > 0 {
			usedQuery = q
			break
		}
	}
	if searchErr != nil && len(candidates) == 0 {
		return nil, 424, searchErr
	}
	if len(candidates) == 0 {
		tlog.Infof("search returned 0 candidates")
		result.Message = "no PMVHaven candidates found"
		return result, 200, nil
	}
	result.Query = usedQuery
	if usedQuery != query {
		tlog.Infof("fallback query selected used_query=%q base_query=%q", usedQuery, query)
	}
	for i, c := range candidates {
		tlog.Infof("parsed candidate #%d title=%q channel=%q scene_url=%q thumbnail_url=%q", i+1, c.Title, c.Channel, c.SceneURL, c.ThumbnailURL)
	}

	thumbCache := map[string]string{}
	for i := range candidates {
		cacheKey := candidates[i].SceneURL
		if cachedThumb, ok := thumbCache[cacheKey]; ok {
			if strings.TrimSpace(candidates[i].ThumbnailURL) == "" && cachedThumb != "" {
				candidates[i].ThumbnailURL = cachedThumb
			}
			continue
		}

		prevThumb := strings.TrimSpace(candidates[i].ThumbnailURL)
		enriched, enrichErr := scrape.EnrichPMVHavenCandidateThumbnail(candidates[i])
		if enrichErr != nil {
			tlog.Warnf("candidate #%d scene-page thumbnail enrichment failed scene_url=%q err=%v", i+1, candidates[i].SceneURL, enrichErr)
			thumbCache[cacheKey] = prevThumb
			continue
		}

		thumbCache[cacheKey] = strings.TrimSpace(enriched.ThumbnailURL)
		candidates[i] = enriched

		source := "search_html"
		if strings.TrimSpace(enriched.ThumbnailURL) != "" && strings.TrimSpace(enriched.ThumbnailURL) != prevThumb {
			source = "scene_html"
		}
		tlog.Infof("candidate #%d thumbnail source=%s thumbnail_url=%q", i+1, source, enriched.ThumbnailURL)
	}

	ranked := scorePMVCandidatesByText(query, candidates)
	if len(ranked) > 0 {
		tlog.Infof("baseline top title=%q pmv_id=%s", ranked[0].Title, ranked[0].PMVID)
	}

	sortCandidates(ranked)

	for i := range ranked {
		ranked[i].Rank = i + 1
	}
	result.Candidates = ranked
	if len(ranked) > 0 {
		result.MatchedSceneID = buildPMVCustomSceneID(fileID)
		tlog.Infof("final top parsed title=%q pmv_id=%s thumbnail=%q", ranked[0].Title, ranked[0].PMVID, ranked[0].ThumbnailURL)
	}

	if dryRun {
		result.Message = "dry run: best candidate found, no database changes applied"
		tlog.Infof("dry run success candidate_title=%q pmv_id=%s", ranked[0].Title, ranked[0].PMVID)
		return result, 200, nil
	}

	matchedSceneID, err := applyPMVMatch(db, &file, ranked[0])
	if err != nil {
		tlog.Errorf("apply match failed candidate_title=%q pmv_id=%s err=%v", ranked[0].Title, ranked[0].PMVID, err)
		return nil, 500, err
	}

	result.Autolinked = true
	result.MatchedSceneID = matchedSceneID
	result.Message = "file linked to custom PMV scene"
	tlog.Infof("autolinked scene_id=%s candidate_title=%q", matchedSceneID, ranked[0].Title)
	return result, 200, nil
}

func normalizePMVQuery(filename string) string {
	name := strings.TrimSpace(filename)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = regexp.MustCompile(`([a-z])([A-Z])`).ReplaceAllString(name, "$1 $2")
	name = strings.ToLower(name)
	// Common downloader/exporter suffixes: "..._<epoch_ms>_<randomid>".
	name = regexp.MustCompile(`[_\-\s]\d{10,}[_\-\s][a-z0-9]{6,}$`).ReplaceAllString(name, "")

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
	name = regexp.MustCompile(`[^a-z0-9\s']+`).ReplaceAllString(name, " ")
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	skipTokens := map[string]bool{
		"1080p": true, "1440p": true, "2160p": true, "4k": true, "5k": true, "6k": true, "8k": true,
		"fps": true, "60fps": true, "30fps": true, "4k60": true,
		"sbs": true, "tb": true, "lr": true, "vr": true,
		"mp4": true, "mkv": true, "avi": true, "mov": true, "wmv": true,
		"uhd": true, "hd": true, "fullhd": true, "hq": true,
		"h264": true, "h265": true, "x264": true, "x265": true, "264": true, "265": true,
		"2880p": true, "4320p": true, "upscale": true,
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

func buildPMVSearchQueries(filename, baseQuery string) []string {
	added := map[string]bool{}
	out := make([]string, 0, 16)
	add := func(q string) {
		q = strings.TrimSpace(strings.Join(strings.Fields(q), " "))
		if q == "" || added[q] {
			return
		}
		added[q] = true
		out = append(out, q)
	}

	add(baseQuery)
	baseTokens := strings.Fields(baseQuery)
	if len(baseTokens) >= 2 {
		add(strings.Join(baseTokens[1:], " "))
	}
	if len(baseTokens) >= 3 {
		add(strings.Join(baseTokens[:len(baseTokens)-1], " "))
	}
	if len(baseTokens) >= 4 {
		add(strings.Join(baseTokens[1:len(baseTokens)-1], " "))
	}

	cleaned := stripPMVSearchNoise(baseTokens)
	if len(cleaned) > 0 {
		add(strings.Join(cleaned, " "))
		if len(cleaned) >= 2 {
			add(strings.Join(cleaned[1:], " "))
		}
	}

	// If uploader/channel noise is appended at the end, progressively trim
	// trailing tokens so we can land on the true title prefix.
	const (
		minTokensForTailTrim = 3
		maxTailTrimVariants  = 12
	)
	tailAdded := 0
	for end := len(baseTokens) - 1; end >= minTokensForTailTrim && tailAdded < maxTailTrimVariants; end-- {
		add(strings.Join(baseTokens[:end], " "))
		tailAdded++
	}
	if len(cleaned) > minTokensForTailTrim {
		cleanedTailAdded := 0
		for end := len(cleaned) - 1; end >= minTokensForTailTrim && cleanedTailAdded < maxTailTrimVariants; end-- {
			add(strings.Join(cleaned[:end], " "))
			cleanedTailAdded++
		}
	}

	raw := strings.TrimSpace(strings.TrimSuffix(filename, filepath.Ext(filename)))
	if raw != "" {
		for _, sep := range []string{"_-_", " - ", " – ", " — "} {
			if strings.Contains(raw, sep) {
				parts := strings.Split(raw, sep)
				if len(parts) >= 2 {
					add(normalizePMVQuery(strings.Join(parts[1:], " ")))
				}
				if len(parts) >= 3 {
					add(normalizePMVQuery(strings.Join(parts[2:], " ")))
				}
			}
		}
	}

	return out
}

func stripPMVSearchNoise(tokens []string) []string {
	if len(tokens) == 0 {
		return nil
	}
	skip := map[string]bool{
		"1080p": true, "1440p": true, "2160p": true, "2880p": true, "4320p": true, "4k": true, "8k": true,
		"60fps": true, "30fps": true, "fps": true,
		"h264": true, "h265": true, "x264": true, "x265": true, "264": true, "265": true,
		"sbs": true, "tb": true, "lr": true, "vr": true,
		"upscale": true, "remaster": true,
	}
	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		tok = strings.TrimSpace(strings.ToLower(tok))
		if tok == "" || skip[tok] {
			continue
		}
		out = append(out, tok)
	}
	return out
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
			Channel:      c.Channel,
			Description:  c.Description,
			Confidence:   confidence,
			Reason:       "baseline text similarity",
		})
	}
	sortCandidates(out)
	return out
}

func inferPMVStudio(filename, candidateTitle, channel string) string {
	if fromChannel := cleanStudioToken(channel); fromChannel != "" {
		return fromChannel
	}

	for _, sep := range []string{" - ", " – ", " — ", "|"} {
		if idx := strings.Index(candidateTitle, sep); idx > 0 {
			prefix := cleanStudioToken(candidateTitle[:idx])
			if prefix != "" {
				return prefix
			}
		}
	}

	raw := strings.TrimSpace(strings.TrimSuffix(filename, filepath.Ext(filename)))
	if raw == "" {
		return ""
	}

	if strings.Contains(raw, "_-_") {
		parts := strings.Split(raw, "_-_")
		for i := 0; i < len(parts) && i < 3; i++ {
			p := cleanStudioToken(parts[i])
			if p == "" {
				continue
			}
			if strings.Contains(strings.ToLower(p), "pmv") {
				return p
			}
			if i == 0 {
				return p
			}
		}
	}

	for _, sep := range []string{" - ", " – ", " — "} {
		if idx := strings.Index(raw, sep); idx > 0 {
			prefix := cleanStudioToken(raw[:idx])
			if prefix != "" {
				return prefix
			}
		}
	}

	return cleanStudioToken(raw)
}

func cleanStudioToken(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "[](){}-_.,:; ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, ".", " ")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return ""
	}
	if len(s) > 64 {
		return ""
	}
	hasLetter := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		return ""
	}
	return s
}

func applyPMVMatch(db *gorm.DB, file *models.File, candidate PMVMatchCandidate) (string, error) {
	sceneID := buildPMVCustomSceneID(file.ID)
	studio := inferPMVStudio(file.Filename, candidate.Title, candidate.Channel)
	if studio == "" {
		studio = "Custom"
	}
	now := time.Now()
	sceneWasCreated := false
	var existing models.Scene
	if err := db.Where(&models.Scene{SceneID: sceneID}).First(&existing).Error; err == gorm.ErrRecordNotFound {
		sceneWasCreated = true
	}

	ext := models.ScrapedScene{
		SceneID:     sceneID,
		ScraperID:   "custom",
		SceneType:   "VR",
		Title:       strings.TrimSpace(candidate.Title),
		Studio:      studio,
		Site:        "CustomVR",
		HomepageURL: strings.TrimSpace(candidate.SceneURL),
		MembersUrl:  strings.TrimSpace(candidate.SceneURL),
		Released:    now.Format("2006-01-02"),
		Filenames:   []string{file.Filename},
		Synopsis:    strings.TrimSpace(candidate.Description),
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
	if sceneWasCreated {
		if err := db.Model(&scene).Updates(map[string]interface{}{
			"created_at": now,
			"added_date": now,
		}).Error; err != nil {
			return "", err
		}
		scene.CreatedAt = now
		scene.AddedDate = now
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

func pmvSceneSourceURL(scene models.Scene) string {
	for _, raw := range []string{scene.SceneURL, scene.MemberURL} {
		u := strings.TrimSpace(raw)
		if u == "" {
			continue
		}
		if strings.Contains(strings.ToLower(u), "pmvhaven.com") {
			return u
		}
	}
	return ""
}

func updatePMVSceneImagesJSON(thumbnailURL string) string {
	images := []models.Image{{
		URL:  strings.TrimSpace(thumbnailURL),
		Type: "cover",
	}}
	b, err := json.Marshal(images)
	if err != nil {
		return ""
	}
	return string(b)
}

func refreshPMVSceneMetadata(scene models.Scene, opts pmvMetadataUpdateOptions) (models.Scene, bool, string, error) {
	sourceURL := pmvSceneSourceURL(scene)
	if sourceURL == "" {
		return scene, false, "no PMVHaven URL on scene", nil
	}

	enriched, err := scrape.EnrichPMVHavenCandidateThumbnail(scrape.PMVHavenCandidate{
		SceneURL: sourceURL,
		Title:    scene.Title,
		Channel:  scene.Studio,
	})
	if err != nil {
		return scene, false, "", err
	}

	changed := false
	updatedFields := make([]string, 0, 5)

	if opts.Title {
		if title := strings.TrimSpace(enriched.Title); title != "" && title != scene.Title {
			scene.Title = title
			changed = true
			updatedFields = append(updatedFields, "title")
		}
	}

	if opts.Studio {
		studio := cleanStudioToken(enriched.Channel)
		if studio != "" && studio != scene.Studio {
			scene.Studio = studio
			changed = true
			updatedFields = append(updatedFields, "studio")
		}
	}

	if opts.SceneURL {
		newURL := strings.TrimSpace(enriched.SceneURL)
		if newURL == "" {
			newURL = sourceURL
		}
		localChanged := false
		if newURL != "" && scene.SceneURL != newURL {
			scene.SceneURL = newURL
			localChanged = true
		}
		if newURL != "" && scene.MemberURL != newURL {
			scene.MemberURL = newURL
			localChanged = true
		}
		if localChanged {
			changed = true
			updatedFields = append(updatedFields, "scene_url")
		}
	}

	if opts.Thumbnail {
		if thumb := strings.TrimSpace(enriched.ThumbnailURL); thumb != "" {
			localChanged := false
			if scene.CoverURL != thumb {
				scene.CoverURL = thumb
				localChanged = true
			}
			if imagesJSON := updatePMVSceneImagesJSON(thumb); imagesJSON != "" && scene.Images != imagesJSON {
				scene.Images = imagesJSON
				localChanged = true
			}
			if localChanged {
				changed = true
				updatedFields = append(updatedFields, "thumbnail")
			}
		}
	}

	if opts.Description {
		if description := strings.TrimSpace(enriched.Description); description != "" && description != scene.Synopsis {
			scene.Synopsis = description
			changed = true
			updatedFields = append(updatedFields, "description")
		}
	}

	return scene, changed, strings.Join(updatedFields, ","), nil
}

func RefreshPMVMatchedScenes(req PMVMatchBatchRequest) (*PMVMatchBatchResult, int, error) {
	concurrency := normalizePMVBatchConcurrency(req.Concurrency)
	opts := normalizePMVMetadataUpdateOptions(req)

	db, _ := models.GetDB()
	defer db.Close()

	var scenes []models.Scene
	if err := db.Model(&models.Scene{}).
		Joins("LEFT JOIN files ON files.scene_id = scenes.id AND files.type = ?", "video").
		Group("scenes.id").
		Order("COALESCE(MAX(files.created_time), scenes.added_date, scenes.created_at) DESC").
		Find(&scenes).Error; err != nil {
		return nil, 500, err
	}

	out := &PMVMatchBatchResult{
		Scanned: len(scenes),
		Mode:    "refresh_metadata",
		Results: make([]PMVMatchBatchItem, len(scenes)),
	}
	if len(scenes) == 0 {
		return out, 200, nil
	}

	if concurrency > len(scenes) {
		concurrency = len(scenes)
	}

	type batchJob struct {
		Index int
		Scene models.Scene
	}
	type batchResult struct {
		Index int
		Item  PMVMatchBatchItem
		Scene *models.Scene
	}

	jobs := make(chan batchJob)
	results := make(chan batchResult, len(scenes))

	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for job := range jobs {
			item := PMVMatchBatchItem{
				SceneID:    job.Scene.SceneID,
				Filename:   job.Scene.Title,
				StatusCode: 200,
			}

			updatedScene, changed, fieldMsg, err := refreshPMVSceneMetadata(job.Scene, opts)
			if err != nil {
				item.StatusCode = 424
				item.Error = err.Error()
				results <- batchResult{Index: job.Index, Item: item}
				continue
			}
			if fieldMsg == "no PMVHaven URL on scene" {
				item.StatusCode = 204
				item.Message = fieldMsg
				results <- batchResult{Index: job.Index, Item: item}
				continue
			}

			if !changed {
				item.Message = "no metadata changes"
				results <- batchResult{Index: job.Index, Item: item}
				continue
			}

			item.Updated = true
			item.Message = "updated fields: " + fieldMsg
			if req.DryRun {
				item.Message = "dry run: " + item.Message
				results <- batchResult{Index: job.Index, Item: item}
				continue
			}

			if err := updatedScene.Save(); err != nil {
				item.StatusCode = 500
				item.Error = err.Error()
				item.Updated = false
				results <- batchResult{Index: job.Index, Item: item}
				continue
			}

			sceneCopy := updatedScene
			results <- batchResult{Index: job.Index, Item: item, Scene: &sceneCopy}
		}
	}

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}

	go func() {
		for i, scene := range scenes {
			jobs <- batchJob{Index: i, Scene: scene}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	changedScenes := make([]models.Scene, 0, len(scenes))
	for r := range results {
		out.Results[r.Index] = r.Item
		if r.Scene != nil {
			changedScenes = append(changedScenes, *r.Scene)
		}
	}

	if len(changedScenes) > 0 {
		IndexScenes(&changedScenes)
	}

	for _, item := range out.Results {
		if item.Error != "" {
			out.Errors++
			continue
		}
		if item.StatusCode == 204 {
			out.SkippedNoPMVURL++
			continue
		}
		if item.Updated {
			out.Updated++
		}
	}

	return out, 200, nil
}

func MatchPMVUnmatchedFiles(req PMVMatchBatchRequest) (*PMVMatchBatchResult, int, error) {
	if req.RefreshExisting {
		return RefreshPMVMatchedScenes(req)
	}

	limit := normalizePMVBatchLimit(req.Limit)
	concurrency := normalizePMVBatchConcurrency(req.Concurrency)

	db, _ := models.GetDB()
	defer db.Close()

	query := db.Model(&models.File{}).Where("type = ? AND scene_id = 0", "video")
	if req.VolumeID != 0 {
		query = query.Where("volume_id = ?", req.VolumeID)
	}
	if strings.TrimSpace(req.PathPrefix) != "" {
		query = query.Where("path LIKE ?", strings.TrimSpace(req.PathPrefix)+"%")
	}

	var files []models.File
	if err := query.Order("created_time desc").Limit(limit).Find(&files).Error; err != nil {
		return nil, 500, err
	}

	out := &PMVMatchBatchResult{
		Scanned: len(files),
		Mode:    "match_unmatched",
		Results: make([]PMVMatchBatchItem, len(files)),
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
		Item  PMVMatchBatchItem
	}

	jobs := make(chan batchJob)
	results := make(chan batchResult, len(files))

	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for job := range jobs {
			item := PMVMatchBatchItem{
				FileID:   job.File.ID,
				Filename: job.File.Filename,
			}

			result, statusCode, err := MatchPMVFile(job.File.ID, req.DryRun)
			if statusCode == 0 {
				statusCode = 500
			}
			item.StatusCode = statusCode
			if err != nil {
				item.Error = err.Error()
				results <- batchResult{Index: job.Index, Item: item}
				continue
			}

			item.Result = result
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

	for r := range results {
		out.Results[r.Index] = r.Item
	}

	for _, item := range out.Results {
		if item.Error != "" {
			if item.StatusCode == 409 {
				out.SkippedAlreadyMatch++
			} else {
				out.Errors++
			}
			continue
		}
		if item.Result == nil {
			out.Errors++
			continue
		}
		if item.Result.Autolinked {
			out.Matched++
		}
	}

	return out, 200, nil
}

func RunPMVMatchUnmatchedTask(req PMVMatchBatchRequest) {
	tlog := log.WithField("task", "pmv-match-unmatched")
	if models.CheckLock("pmv-match") {
		tlog.Infof("skipped: task already running")
		return
	}

	models.CreateLock("pmv-match")
	defer models.RemoveLock("pmv-match")

	if req.RefreshExisting {
		opts := normalizePMVMetadataUpdateOptions(req)
		tlog.Infof("start mode=refresh_metadata dry_run=%v concurrency=%d update_fields=%q",
			req.DryRun, normalizePMVBatchConcurrency(req.Concurrency), opts.String())
	} else {
		tlog.Infof("start mode=match_unmatched dry_run=%v limit=%d concurrency=%d volume_id=%d path_prefix=%q",
			req.DryRun, req.Limit, normalizePMVBatchConcurrency(req.Concurrency), req.VolumeID, req.PathPrefix)
	}
	result, statusCode, err := MatchPMVUnmatchedFiles(req)
	if err != nil {
		tlog.Errorf("failed status=%d err=%v", statusCode, err)
		return
	}

	if result.Mode == "refresh_metadata" {
		tlog.Infof("done status=%d mode=%s scanned=%d updated=%d skipped_no_pmv_url=%d errors=%d",
			statusCode, result.Mode, result.Scanned, result.Updated, result.SkippedNoPMVURL, result.Errors)
		return
	}
	tlog.Infof("done status=%d mode=%s scanned=%d matched=%d skipped_already_matched=%d errors=%d",
		statusCode, result.Mode, result.Scanned, result.Matched, result.SkippedAlreadyMatch, result.Errors)
}

func normalizePMVBatchLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func normalizePMVBatchConcurrency(concurrency int) int {
	if concurrency <= 0 {
		return 10
	}
	if concurrency > 50 {
		return 50
	}
	return concurrency
}

func buildPMVCustomSceneID(fileID uint) string {
	return fmt.Sprintf("custom-pmv-%d", fileID)
}
