package scrape

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

const pmvHavenBaseURL = "https://pmvhaven.com"

type PMVHavenCandidate struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	SceneURL     string `json:"scene_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Channel      string `json:"channel,omitempty"`
	Description  string `json:"description,omitempty"`
}

type PMVHavenVideoMetadata struct {
	SceneURL     string `json:"scene_url"`
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnail_url"`
	Channel      string `json:"channel,omitempty"`
	Description  string `json:"description,omitempty"`
	MediaURL     string `json:"media_url"`
	Filename     string `json:"filename"`
}

func FetchPMVHavenVideoMetadata(sceneURL string) (PMVHavenVideoMetadata, error) {
	sceneURL = canonicalSceneURL(sceneURL)
	if sceneURL == "" {
		return PMVHavenVideoMetadata{}, fmt.Errorf("invalid PMVHaven URL")
	}

	client := resty.New().
		SetTimeout(25*time.Second).
		SetRetryCount(2).
		SetHeader("User-Agent", UserAgent)

	req := client.R()
	SetupRestyRequest("pmvhaven-scraper", req)

	resp, err := req.Get(sceneURL)
	if err != nil {
		return PMVHavenVideoMetadata{}, err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return PMVHavenVideoMetadata{}, fmt.Errorf("pmvhaven scene fetch failed with status %d", resp.StatusCode())
	}

	htmlBody := resp.String()
	mediaURL := ParsePMVHavenSceneHTMLForMediaURL(htmlBody)
	if mediaURL == "" {
		return PMVHavenVideoMetadata{}, fmt.Errorf("no downloadable media URL found on PMVHaven page")
	}

	filename := filepath.Base(mediaURL)
	if filename == "." || filename == "/" || filename == "" {
		return PMVHavenVideoMetadata{}, fmt.Errorf("could not determine filename from media URL")
	}

	return PMVHavenVideoMetadata{
		SceneURL:     sceneURL,
		Title:        ParsePMVHavenSceneHTMLForTitle(htmlBody),
		ThumbnailURL: ParsePMVHavenSceneHTMLForThumbnail(htmlBody),
		Channel:      ParsePMVHavenSceneHTMLForChannel(htmlBody),
		Description:  ParsePMVHavenSceneHTMLForDescription(htmlBody),
		MediaURL:     mediaURL,
		Filename:     filename,
	}, nil
}

func EnrichPMVHavenCandidateThumbnail(c PMVHavenCandidate) (PMVHavenCandidate, error) {
	sceneURL := canonicalSceneURL(c.SceneURL)
	if sceneURL == "" {
		return c, fmt.Errorf("invalid scene url")
	}

	client := resty.New().
		SetTimeout(25*time.Second).
		SetRetryCount(2).
		SetHeader("User-Agent", UserAgent)

	req := client.R()
	SetupRestyRequest("pmvhaven-scraper", req)

	resp, err := req.Get(sceneURL)
	if err != nil {
		return c, err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return c, fmt.Errorf("pmvhaven scene fetch failed with status %d", resp.StatusCode())
	}

	if thumb := ParsePMVHavenSceneHTMLForThumbnail(resp.String()); thumb != "" {
		c.ThumbnailURL = thumb
	}
	if title := ParsePMVHavenSceneHTMLForTitle(resp.String()); title != "" {
		c.Title = title
	}
	if channel := ParsePMVHavenSceneHTMLForChannel(resp.String()); channel != "" {
		c.Channel = channel
	}
	if description := ParsePMVHavenSceneHTMLForDescription(resp.String()); description != "" {
		c.Description = description
	}
	return c, nil
}

func ParsePMVHavenSceneHTMLForThumbnail(htmlBody string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return ""
	}

	thumb := strings.TrimSpace(firstNonEmpty(
		attrVal(doc.Find(`meta[property="og:image"]`).First(), "content"),
		attrVal(doc.Find(`meta[name="twitter:image"]`).First(), "content"),
		attrVal(doc.Find(`video[poster]`).First(), "poster"),
	))
	if thumb != "" {
		return absoluteURL(thumb)
	}

	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, script *goquery.Selection) bool {
		text := strings.TrimSpace(script.Text())
		if text == "" {
			return true
		}
		thumb = parseJSONLDThumbnail(text)
		return thumb == ""
	})
	if thumb != "" {
		return absoluteURL(thumb)
	}
	return ""
}

func ParsePMVHavenSceneHTMLForTitle(htmlBody string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return ""
	}

	title := strings.TrimSpace(firstNonEmpty(
		attrVal(doc.Find(`meta[property="og:title"]`).First(), "content"),
		attrVal(doc.Find(`meta[name="twitter:title"]`).First(), "content"),
	))
	if title == "" {
		title = strings.TrimSpace(doc.Find("title").First().Text())
	}
	return cleanPMVHavenTitle(title)
}

func ParsePMVHavenSceneHTMLForChannel(htmlBody string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return ""
	}

	channel := cleanPMVHavenChannel(firstNonEmpty(
		firstText(doc.Selection,
			`a[rel="author"]`,
			`.author a`,
			`.entry-author a`,
			`.byline a`,
			`a[href*="/author/"]`,
		),
		channelFromAuthorURL(firstAttr(doc.Selection,
			`a[rel="author"][href]`,
			`.author a[href]`,
			`.entry-author a[href]`,
			`.byline a[href]`,
			`a[href*="/author/"][href]`,
		)),
		channelFromAuthorURL(attrVal(doc.Find(`meta[property="article:author"]`).First(), "content")),
	))
	if channel != "" {
		return channel
	}

	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, script *goquery.Selection) bool {
		text := strings.TrimSpace(script.Text())
		if text == "" {
			return true
		}
		channel = parseJSONLDChannel(text)
		return channel == ""
	})
	return cleanPMVHavenChannel(channel)
}

func ParsePMVHavenSceneHTMLForDescription(htmlBody string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return ""
	}

	best := cleanPMVHavenDescription(firstNonEmpty(
		attrVal(doc.Find(`meta[property="og:description"]`).First(), "content"),
		attrVal(doc.Find(`meta[name="description"]`).First(), "content"),
		attrVal(doc.Find(`meta[name="twitter:description"]`).First(), "content"),
	))

	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, script *goquery.Selection) bool {
		text := strings.TrimSpace(script.Text())
		if text == "" {
			return true
		}
		if desc := parseJSONLDDescription(text); len(desc) > len(best) {
			best = desc
		}
		return true
	})

	re := regexp.MustCompile(`"description":"((?:\\.|[^"\\])*)"`)
	matches := re.FindAllStringSubmatch(htmlBody, -1)
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		desc := match[1]
		if unquoted, err := strconv.Unquote(`"` + desc + `"`); err == nil {
			desc = unquoted
		}
		desc = cleanPMVHavenDescription(desc)
		if len(desc) > len(best) {
			best = desc
		}
	}

	return best
}

func ParsePMVHavenSceneHTMLForMediaURL(htmlBody string) string {
	if direct := parsePMVHavenSceneJSONLDMediaURL(htmlBody); direct != "" {
		return direct
	}
	if direct := parsePMVHavenSceneNuxtMediaURL(htmlBody); direct != "" {
		return direct
	}

	candidates := make([]string, 0, 8)
	addCandidate := func(raw string) {
		raw = normalizePMVHavenMediaURL(raw)
		if raw == "" {
			return
		}
		candidates = append(candidates, raw)
	}

	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`"videoUrl":"((?:\\.|[^"\\])+)"`),
		regexp.MustCompile(`"contentUrl":"((?:\\.|[^"\\])+)"`),
		regexp.MustCompile(`https:\\u002F\\u002Fvideo\.pmvhaven\.com\\u002Fvideos\\u002F[^"'<\s]+\.mp4`),
		regexp.MustCompile(`https://video\.pmvhaven\.com/videos/[^"'\\<\s]+\.mp4`),
	} {
		matches := re.FindAllStringSubmatch(htmlBody, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				addCandidate(match[1])
			} else if len(match) == 1 {
				addCandidate(match[0])
			}
		}
	}

	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		return candidate
	}
	return ""
}

func parsePMVHavenSceneJSONLDMediaURL(htmlBody string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return ""
	}

	currentSceneURL := parsePMVHavenSceneCanonicalURL(doc)
	currentTitle := cleanPMVHavenTitle(firstNonEmpty(
		attrVal(doc.Find(`meta[property="og:title"]`).First(), "content"),
		attrVal(doc.Find(`meta[name="twitter:title"]`).First(), "content"),
		strings.TrimSpace(doc.Find("title").First().Text()),
	))

	fallback := ""
	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, script *goquery.Selection) bool {
		text := strings.TrimSpace(script.Text())
		if text == "" {
			return true
		}
		for _, candidate := range parseJSONLDVideoMediaCandidates(text) {
			mediaURL := normalizePMVHavenMediaURL(candidate.MediaURL)
			if mediaURL == "" {
				continue
			}
			if candidate.SceneURL != "" && currentSceneURL != "" && candidate.SceneURL == currentSceneURL {
				fallback = mediaURL
				return false
			}
			if candidate.Title != "" && currentTitle != "" && cleanPMVHavenTitle(candidate.Title) == currentTitle {
				fallback = mediaURL
				return false
			}
			if fallback == "" {
				fallback = mediaURL
			}
		}
		return true
	})

	return fallback
}

func parsePMVHavenSceneNuxtMediaURL(htmlBody string) string {
	scriptBody := extractPMVHavenNuxtDataScript(htmlBody)
	if strings.TrimSpace(scriptBody) == "" {
		return ""
	}

	var root []interface{}
	if err := json.Unmarshal([]byte(scriptBody), &root); err != nil {
		return ""
	}

	currentTitle := cleanPMVHavenTitle(ParsePMVHavenSceneHTMLForTitle(htmlBody))
	fallback := ""
	matched := false

	var walk func(interface{})
	walk = func(node interface{}) {
		if matched {
			return
		}
		switch val := node.(type) {
		case []interface{}:
			for _, child := range val {
				walk(child)
				if matched {
					return
				}
			}
		case map[string]interface{}:
			videoNode := val
			if rawVideo, ok := val["video"]; ok {
				if resolved, ok := resolvePMVHavenNuxtValue(root, rawVideo, map[int]bool{}).(map[string]interface{}); ok {
					videoNode = resolved
				}
			}

			title := cleanPMVHavenTitle(resolvePMVHavenNuxtString(root, videoNode["title"]))
			mediaURL := normalizePMVHavenMediaURL(resolvePMVHavenNuxtString(root, videoNode["videoUrl"]))
			if mediaURL != "" {
				if title != "" && currentTitle != "" && title == currentTitle {
					fallback = mediaURL
					matched = true
					return
				}
				if fallback == "" {
					fallback = mediaURL
				}
			}

			for _, child := range val {
				walk(child)
				if matched {
					return
				}
			}
		}
	}

	walk(root)
	return fallback
}

type pmvJSONLDMediaCandidate struct {
	SceneURL string
	Title    string
	MediaURL string
}

func parseJSONLDVideoMediaCandidates(data string) []pmvJSONLDMediaCandidate {
	root := gjson.Parse(data)
	out := make([]pmvJSONLDMediaCandidate, 0, 2)

	var visit func(gjson.Result)
	visit = func(node gjson.Result) {
		if !node.Exists() {
			return
		}
		if node.IsArray() {
			for _, child := range node.Array() {
				visit(child)
			}
			return
		}
		if !node.IsObject() {
			return
		}

		if strings.EqualFold(node.Get("@type").String(), "VideoObject") {
			sceneURL := canonicalSceneURL(firstNonEmpty(
				strings.TrimSpace(node.Get("url").String()),
				strings.TrimSpace(node.Get("embedUrl").String()),
			))
			mediaURL := strings.TrimSpace(node.Get("contentUrl").String())
			title := strings.TrimSpace(node.Get("name").String())
			if mediaURL != "" {
				out = append(out, pmvJSONLDMediaCandidate{
					SceneURL: sceneURL,
					Title:    title,
					MediaURL: mediaURL,
				})
			}
		}

		node.ForEach(func(_, child gjson.Result) bool {
			visit(child)
			return true
		})
	}

	visit(root)
	return out
}

func normalizePMVHavenMediaURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if unquoted, err := strconv.Unquote(`"` + raw + `"`); err == nil {
		raw = unquoted
	}
	raw = strings.ReplaceAll(raw, `\u002F`, `/`)
	raw = html.UnescapeString(raw)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if strings.Contains(strings.ToLower(raw), ".mp4/master.m3u8") {
		raw = raw[:strings.Index(strings.ToLower(raw), ".mp4/master.m3u8")+4]
	}

	lower := strings.ToLower(raw)
	if !strings.HasSuffix(lower, ".mp4") {
		return ""
	}
	if strings.Contains(lower, "_preview.mp4") {
		return ""
	}
	if strings.Contains(lower, "/previews/") || strings.Contains(lower, "/timeline-thumbnails/") {
		return ""
	}
	return raw
}

func parsePMVHavenSceneCanonicalURL(doc *goquery.Document) string {
	if doc == nil {
		return ""
	}
	return canonicalSceneURL(firstNonEmpty(
		attrVal(doc.Find(`link[rel="canonical"]`).First(), "href"),
		attrVal(doc.Find(`meta[property="og:url"]`).First(), "content"),
	))
}

func SearchPMVHaven(query string, limit int) ([]PMVHavenCandidate, error) {
	q := url.QueryEscape(strings.TrimSpace(query))
	searchURLs := []string{
		fmt.Sprintf("%s/search?q=%s", pmvHavenBaseURL, q),
	}
	tlog := log.WithField("task", "pmvhaven-scraper")

	client := resty.New().
		SetTimeout(25*time.Second).
		SetRetryCount(2).
		SetHeader("User-Agent", UserAgent)

	var lastErr error
	seen := map[string]bool{}
	allCandidates := make([]PMVHavenCandidate, 0, limit)
	for idx, searchURL := range searchURLs {
		tlog.Infof("call #%d query=%q url=%s", idx+1, query, searchURL)
		req := client.R()
		SetupRestyRequest("pmvhaven-scraper", req)

		resp, err := req.Get(searchURL)
		if err != nil {
			tlog.Warnf("call #%d failed url=%s err=%v", idx+1, searchURL, err)
			lastErr = err
			continue
		}
		tlog.Infof("call #%d response status=%d bytes=%d url=%s", idx+1, resp.StatusCode(), len(resp.String()), searchURL)
		if dumpPath, dumpErr := dumpPMVHavenHTML(query, idx+1, searchURL, resp.String()); dumpErr != nil {
			tlog.Warnf("call #%d html dump failed url=%s err=%v", idx+1, searchURL, dumpErr)
		} else {
			tlog.Infof("call #%d html dump file=%s", idx+1, dumpPath)
		}
		if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
			lastErr = fmt.Errorf("pmvhaven search failed with status %d", resp.StatusCode())
			continue
		}

		candidates := ParsePMVHavenSearchHTML(resp.String(), limit)
		tlog.Infof("call #%d parsed_candidates=%d url=%s", idx+1, len(candidates), searchURL)
		for i, c := range candidates {
			tlog.Infof("call #%d candidate #%d title=%q scene_url=%q thumbnail_url=%q", idx+1, i+1, c.Title, c.SceneURL, c.ThumbnailURL)
		}
		for _, c := range candidates {
			if len(allCandidates) >= limit {
				break
			}
			if seen[c.SceneURL] {
				continue
			}
			seen[c.SceneURL] = true
			allCandidates = append(allCandidates, c)
		}
		if len(allCandidates) >= limit {
			break
		}
	}

	if len(allCandidates) > 0 {
		tlog.Infof("final candidates=%d query=%q", len(allCandidates), query)
		return allCandidates, nil
	}
	if lastErr != nil {
		tlog.Warnf("no candidates query=%q last_err=%v", query, lastErr)
		return nil, lastErr
	}
	tlog.Infof("no candidates query=%q", query)
	return []PMVHavenCandidate{}, nil
}

func FetchPMVHavenListingCandidates(listURL string, limit int) ([]PMVHavenCandidate, error) {
	listURL = strings.TrimSpace(absoluteURL(listURL))
	if listURL == "" {
		return nil, fmt.Errorf("invalid PMVHaven list URL")
	}
	u, err := url.Parse(listURL)
	if err != nil || !strings.Contains(strings.ToLower(u.Host), "pmvhaven.com") {
		return nil, fmt.Errorf("invalid PMVHaven list URL")
	}

	client := resty.New().
		SetTimeout(25*time.Second).
		SetRetryCount(2).
		SetHeader("User-Agent", UserAgent)

	all := make([]PMVHavenCandidate, 0, maxInt(limit, 32))
	seen := map[string]bool{}
	pageLimit := limit
	if pageLimit <= 0 {
		pageLimit = 0
	}

	appendCandidates := func(candidates []PMVHavenCandidate) {
		for _, candidate := range candidates {
			candidate.SceneURL = canonicalSceneURL(candidate.SceneURL)
			key := pmvCandidateDedupKey(candidate)
			if candidate.SceneURL == "" || key == "" || seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, candidate)
			if limit > 0 && len(all) >= limit {
				return
			}
		}
	}

	firstPageSignature := ""
	for page := 1; page <= 50; page++ {
		pageURL := buildPMVHavenListPageURL(listURL, page)
		req := client.R()
		SetupRestyRequest("pmvhaven-scraper", req)

		resp, err := req.Get(pageURL)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			break
		}
		if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
			if page == 1 {
				return nil, fmt.Errorf("pmvhaven list fetch failed with status %d", resp.StatusCode())
			}
			break
		}

		remaining := pageLimit
		if limit > 0 {
			remaining = limit - len(all)
			if remaining <= 0 {
				break
			}
		}
		candidates := ParsePMVHavenSearchHTML(resp.String(), remaining)
		if len(candidates) == 0 {
			break
		}

		signature := pmvCandidatePageSignature(candidates)
		if page == 1 {
			firstPageSignature = signature
		} else if signature == "" || signature == firstPageSignature {
			break
		}

		before := len(all)
		appendCandidates(candidates)
		if len(all) == before {
			break
		}
		if limit > 0 && len(all) >= limit {
			break
		}
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no PMVHaven videos found on list page")
	}
	return all, nil
}

func dumpPMVHavenHTML(query string, callNum int, callURL string, body string) (string, error) {
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("empty body")
	}
	dir := ".tmp_pmv_debug"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	kind := "search"
	if strings.Contains(callURL, "/?s=") {
		kind = "s-param"
	}
	stamp := time.Now().Format("20060102_150405")
	name := fmt.Sprintf("%s_%02d_%s_%s.html", stamp, callNum, kind, slugForFilename(query))
	fullPath := filepath.Join(dir, name)
	if err := os.WriteFile(fullPath, []byte(body), 0644); err != nil {
		return "", err
	}
	return fullPath, nil
}

func slugForFilename(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "query"
	}
	if len(s) > 80 {
		return s[:80]
	}
	return s
}

func ParsePMVHavenSearchHTML(htmlBody string, limit int) []PMVHavenCandidate {
	if limit <= 0 {
		limit = 10000
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return []PMVHavenCandidate{}
	}

	seen := map[string]bool{}
	out := make([]PMVHavenCandidate, 0, limit)
	addCandidate := func(c PMVHavenCandidate) bool {
		c.SceneURL = canonicalSceneURL(c.SceneURL)
		c.ThumbnailURL = absoluteURL(c.ThumbnailURL)
		c.Title = strings.TrimSpace(c.Title)
		if c.ID == "" {
			c.ID = buildCandidateID(c.SceneURL)
		}
		key := pmvCandidateDedupKey(c)
		if c.SceneURL == "" || c.Title == "" || key == "" || seen[key] {
			return false
		}
		seen[key] = true
		out = append(out, c)
		return len(out) >= limit
	}

	profileOwner := ""
	profileTotal := 0
	if isPMVHavenProfilePage(doc) {
		profileOwner = parsePMVHavenProfileOwner(doc)
		profileTotal = parsePMVHavenProfileVideoTotal(doc)
		doc.Find(`.videos-grid-fixed a[href*="/video/"], a[data-video-id][href*="/video/"]`).EachWithBreak(func(_ int, a *goquery.Selection) bool {
			sceneURL := attrVal(a, "href")
			if !looksLikeSceneURL(sceneURL) {
				return true
			}

			title := strings.TrimSpace(cleanPMVHavenTitle(firstNonEmpty(
				attrVal(a, "title"),
				attrVal(a, "aria-label"),
				firstText(a,
					`h1`,
					`h2`,
					`h3`,
					`.line-clamp-2`,
				),
				firstNonEmpty(
					attrVal(a.Find("img").First(), "alt"),
					attrVal(a.Find("img").First(), "title"),
				),
			)))
			if title == "" {
				title = cleanPMVHavenListingTitle("", sceneURL)
			}

			thumbnailURL := firstNonEmpty(
				attrVal(a.Find("img").First(), "data-src"),
				attrVal(a.Find("img").First(), "data-lazy-src"),
				attrVal(a.Find("img").First(), "data-original"),
				attrVal(a.Find("img").First(), "src"),
				attrVal(a.Find("source").First(), "data-srcset"),
				attrVal(a.Find("source").First(), "srcset"),
			)
			if strings.Contains(thumbnailURL, ",") {
				thumbnailURL = strings.TrimSpace(strings.Split(thumbnailURL, ",")[0])
				thumbnailURL = strings.TrimSpace(strings.Split(thumbnailURL, " ")[0])
			}

			channel := cleanPMVHavenChannel(firstNonEmpty(
				firstText(a,
					`[role="link"] span.truncate`,
					`span.truncate`,
				),
				profileOwner,
			))

			if addCandidate(PMVHavenCandidate{
				ID:           buildCandidateID(sceneURL),
				Title:        title,
				SceneURL:     sceneURL,
				ThumbnailURL: thumbnailURL,
				Channel:      channel,
			}) {
				return false
			}
			return true
		})
	}

	if len(out) >= limit {
		return out
	}

	parseCard := func(card *goquery.Selection) bool {
		sceneURL := firstAttr(card,
			`a.entry-title[href]`,
			`h1 a[href]`,
			`h2 a[href]`,
			`h3 a[href]`,
			`a[rel="bookmark"][href]`,
			`a[href]`,
		)
		if !looksLikeSceneURL(sceneURL) {
			return false
		}

		title := cleanPMVHavenListingTitle(firstText(card,
			`.entry-title`,
			`h1`,
			`h2`,
			`h3`,
			`a[rel="bookmark"]`,
			`a`,
		), sceneURL)

		thumbnailURL := firstAttr(card,
			`img[data-src]`,
			`img[data-lazy-src]`,
			`img[data-original]`,
			`img[src]`,
			`source[data-srcset]`,
			`source[srcset]`,
		)
		if strings.Contains(thumbnailURL, ",") {
			thumbnailURL = strings.TrimSpace(strings.Split(thumbnailURL, ",")[0])
			thumbnailURL = strings.TrimSpace(strings.Split(thumbnailURL, " ")[0])
		}

		c := PMVHavenCandidate{
			ID:           buildCandidateID(sceneURL),
			Title:        title,
			SceneURL:     sceneURL,
			ThumbnailURL: thumbnailURL,
			Channel:      extractPMVHavenChannel(card),
		}
		return addCandidate(c)
	}

	// Nuxt pages use /video/* links and inline thumbnails.
	doc.Find(`a[href*="/video/"]`).EachWithBreak(func(_ int, a *goquery.Selection) bool {
		sceneURL, _ := a.Attr("href")
		sceneURL = html.UnescapeString(strings.TrimSpace(sceneURL))
		if !looksLikeSceneURL(sceneURL) {
			return true
		}

		title := cleanPMVHavenListingTitle(firstNonEmpty(
			attrVal(a, "title"),
			attrVal(a, "aria-label"),
			strings.TrimSpace(a.Text()),
		), sceneURL)

		img := a.Find("img").First()
		if img.Length() == 0 {
			// Fallback: some layouts place image in adjacent wrappers.
			img = a.Parent().Find("img").First()
		}

		if title == "" && img.Length() > 0 {
			title = cleanPMVHavenListingTitle(firstNonEmpty(
				attrVal(img, "alt"),
				attrVal(img, "title"),
			), sceneURL)
		}
		if title == "" {
			title = titleFromSceneURL(sceneURL)
		}

		thumbnailURL := ""
		if img.Length() > 0 {
			thumbnailURL = strings.TrimSpace(firstNonEmpty(
				attrVal(img, "data-src"),
				attrVal(img, "data-lazy-src"),
				attrVal(img, "data-original"),
				attrVal(img, "src"),
			))
		}

		c := PMVHavenCandidate{
			ID:           buildCandidateID(sceneURL),
			Title:        title,
			SceneURL:     sceneURL,
			ThumbnailURL: thumbnailURL,
			Channel: func() string {
				channelCtx := a.Closest(`article, .post, .entry, .result-item, .search-result, .type-post`)
				if channelCtx == nil || channelCtx.Length() == 0 {
					channelCtx = a.Parent()
				}
				return extractPMVHavenChannel(channelCtx)
			}(),
		}
		if addCandidate(c) {
			return false
		}
		return true
	})

	if len(out) >= limit {
		return out
	}

	containers := []string{
		`article`,
		`.post`,
		`.entry`,
		`.result-item`,
		`.search-result`,
		`.type-post`,
	}
	for _, sel := range containers {
		stop := false
		doc.Find(sel).EachWithBreak(func(_ int, card *goquery.Selection) bool {
			if parseCard(card) {
				stop = true
				return false
			}
			return true
		})
		if stop || len(out) >= limit {
			break
		}
	}

	if len(out) < limit {
		for _, c := range parsePMVHavenNuxtDataCandidates(htmlBody, profileOwner) {
			if addCandidate(c) {
				return out
			}
		}
	}

	if len(out) < limit {
		doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, script *goquery.Selection) bool {
			text := strings.TrimSpace(script.Text())
			if text == "" {
				return true
			}
			for _, c := range parseJSONLDCandidates(text) {
				if addCandidate(c) {
					return false
				}
			}
			return len(out) < limit
		})
	}

	if profileTotal > 0 && len(out) > profileTotal {
		out = out[:profileTotal]
	}

	return out
}

func buildPMVHavenListPageURL(listURL string, page int) string {
	if page <= 1 {
		return listURL
	}
	u, err := url.Parse(listURL)
	if err != nil {
		return listURL
	}
	q := u.Query()
	q.Set("page", strconv.Itoa(page))
	u.RawQuery = q.Encode()
	return u.String()
}

func pmvCandidatePageSignature(candidates []PMVHavenCandidate) string {
	if len(candidates) == 0 {
		return ""
	}
	parts := make([]string, 0, minInt(len(candidates), 8))
	for i := 0; i < len(candidates) && i < 8; i++ {
		parts = append(parts, canonicalSceneURL(candidates[i].SceneURL))
	}
	return strings.Join(parts, "|")
}

func pmvCandidateDedupKey(candidate PMVHavenCandidate) string {
	if id := strings.TrimSpace(strings.ToLower(candidate.ID)); id != "" {
		return "id:" + id
	}
	sceneURL := canonicalSceneURL(candidate.SceneURL)
	if sceneURL == "" {
		return ""
	}
	return "url:" + sceneURL
}

func parsePMVHavenNuxtDataCandidates(htmlBody, profileOwner string) []PMVHavenCandidate {
	scriptBody := extractPMVHavenNuxtDataScript(htmlBody)
	if strings.TrimSpace(scriptBody) == "" {
		return nil
	}

	var root []interface{}
	if err := json.Unmarshal([]byte(scriptBody), &root); err != nil {
		return nil
	}

	out := make([]PMVHavenCandidate, 0, 64)
	seen := map[string]bool{}

	var walk func(interface{})
	walk = func(node interface{}) {
		switch val := node.(type) {
		case []interface{}:
			for _, child := range val {
				walk(child)
			}
		case map[string]interface{}:
			id := resolvePMVHavenNuxtString(root, val["_id"])
			title := cleanPMVHavenTitle(resolvePMVHavenNuxtString(root, val["title"]))
			videoURL := resolvePMVHavenNuxtString(root, val["videoUrl"])
			thumbnailURL := resolvePMVHavenNuxtString(root, val["thumbnailUrl"])
			channel := cleanPMVHavenChannel(resolvePMVHavenNuxtString(root, val["uploaderUsername"]))
			if profileOwner != "" && !strings.EqualFold(channel, profileOwner) {
				goto descend
			}

			if id != "" && title != "" && videoURL != "" {
				sceneURL := canonicalSceneURL(buildPMVHavenSceneURLFromTitleID(title, id))
				if sceneURL != "" && !seen[sceneURL] {
					seen[sceneURL] = true
					out = append(out, PMVHavenCandidate{
						ID:           id,
						Title:        title,
						SceneURL:     sceneURL,
						ThumbnailURL: absoluteURL(thumbnailURL),
						Channel:      channel,
					})
				}
			}

		descend:
			for _, child := range val {
				walk(child)
			}
		}
	}
	walk(root)
	return out
}

func extractPMVHavenNuxtDataScript(htmlBody string) string {
	const marker = `<script type="application/json" data-nuxt-data="nuxt-app" data-ssr="true" id="__NUXT_DATA__">`
	start := strings.Index(htmlBody, marker)
	if start == -1 {
		return ""
	}
	start += len(marker)
	end := strings.Index(htmlBody[start:], "</script>")
	if end == -1 {
		return ""
	}
	return htmlBody[start : start+end]
}

func resolvePMVHavenNuxtString(root []interface{}, value interface{}) string {
	resolved := resolvePMVHavenNuxtValue(root, value, map[int]bool{})
	if resolved == nil {
		return ""
	}
	switch val := resolved.(type) {
	case string:
		return html.UnescapeString(strings.TrimSpace(val))
	case fmt.Stringer:
		return html.UnescapeString(strings.TrimSpace(val.String()))
	}
	return ""
}

func resolvePMVHavenNuxtValue(root []interface{}, value interface{}, seen map[int]bool) interface{} {
	switch val := value.(type) {
	case float64:
		index := int(val)
		if float64(index) != val || index < 0 || index >= len(root) || seen[index] {
			return value
		}
		seen[index] = true
		return resolvePMVHavenNuxtValue(root, root[index], seen)
	default:
		return value
	}
}

func buildPMVHavenSceneURLFromTitleID(title, id string) string {
	title = strings.TrimSpace(title)
	id = strings.TrimSpace(strings.ToLower(id))
	if title == "" || id == "" {
		return ""
	}
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, "&", " and ")
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return ""
	}
	return fmt.Sprintf("%s/video/%s_%s", pmvHavenBaseURL, slug, id)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func isPMVHavenProfilePage(doc *goquery.Document) bool {
	if doc == nil {
		return false
	}
	pageType := strings.ToLower(strings.TrimSpace(attrVal(doc.Find(`meta[property="og:type"]`).First(), "content")))
	if pageType == "profile" {
		return true
	}
	title := strings.TrimSpace(doc.Find("title").First().Text())
	return strings.Contains(strings.ToLower(title), "profile - pmvhaven")
}

func parsePMVHavenProfileOwner(doc *goquery.Document) string {
	if doc == nil {
		return ""
	}
	title := strings.TrimSpace(html.UnescapeString(doc.Find("title").First().Text()))
	if match := regexp.MustCompile(`(?i)^(.+?)['’]s profile\s*-\s*pmvhaven$`).FindStringSubmatch(title); len(match) == 2 {
		return cleanPMVHavenChannel(match[1])
	}
	desc := strings.TrimSpace(html.UnescapeString(attrVal(doc.Find(`meta[name="description"]`).First(), "content")))
	if match := regexp.MustCompile(`^(.+?)\s+on\s+PMVHaven\b`).FindStringSubmatch(desc); len(match) == 2 {
		return cleanPMVHavenChannel(match[1])
	}
	return ""
}

func parsePMVHavenProfileVideoTotal(doc *goquery.Document) int {
	if doc == nil {
		return 0
	}
	for _, text := range []string{
		attrVal(doc.Find(`meta[name="description"]`).First(), "content"),
		attrVal(doc.Find(`meta[property="og:description"]`).First(), "content"),
		strings.TrimSpace(doc.Find("title").First().Text()),
	} {
		match := regexp.MustCompile(`\b(\d+)\s+videos\s+uploaded\b`).FindStringSubmatch(strings.ToLower(text))
		if len(match) != 2 {
			continue
		}
		total, err := strconv.Atoi(match[1])
		if err == nil && total > 0 {
			return total
		}
	}
	return 0
}

func parseJSONLDCandidates(data string) []PMVHavenCandidate {
	out := []PMVHavenCandidate{}
	seen := map[string]bool{}

	appendCandidate := func(title, sceneURL, thumbnailURL, channel string) {
		sceneURL = canonicalSceneURL(sceneURL)
		if sceneURL == "" || seen[sceneURL] || !looksLikeSceneURL(sceneURL) {
			return
		}
		seen[sceneURL] = true
		out = append(out, PMVHavenCandidate{
			ID:           buildCandidateID(sceneURL),
			Title:        strings.TrimSpace(title),
			SceneURL:     sceneURL,
			ThumbnailURL: absoluteURL(thumbnailURL),
			Channel:      cleanPMVHavenChannel(channel),
		})
	}

	root := gjson.Parse(data)
	visit := func(node gjson.Result, fn func(gjson.Result)) {}
	visit = func(node gjson.Result, fn func(gjson.Result)) {
		if !node.Exists() {
			return
		}
		fn(node)
		if node.IsArray() {
			for _, child := range node.Array() {
				visit(child, fn)
			}
			return
		}
		if node.IsObject() {
			node.ForEach(func(_, child gjson.Result) bool {
				visit(child, fn)
				return true
			})
		}
	}

	visit(root, func(node gjson.Result) {
		if !node.IsObject() {
			return
		}

		title := strings.TrimSpace(node.Get("name").String())
		sceneURL := strings.TrimSpace(node.Get("url").String())
		if title == "" && sceneURL == "" {
			return
		}

		thumbnailURL := strings.TrimSpace(node.Get("thumbnailUrl").String())
		if thumbnailURL == "" {
			thumbnailURL = strings.TrimSpace(node.Get("image.url").String())
		}
		if thumbnailURL == "" {
			thumbnailURL = strings.TrimSpace(node.Get("image").String())
		}
		channel := firstNonEmpty(
			strings.TrimSpace(node.Get("author.name").String()),
			strings.TrimSpace(node.Get("author").String()),
			strings.TrimSpace(node.Get("creator.name").String()),
			strings.TrimSpace(node.Get("creator").String()),
		)
		appendCandidate(title, sceneURL, thumbnailURL, channel)
	})

	return out
}

func parseJSONLDThumbnail(data string) string {
	root := gjson.Parse(data)
	thumb := ""

	var visit func(node gjson.Result)
	visit = func(node gjson.Result) {
		if thumb != "" || !node.Exists() {
			return
		}
		if node.IsObject() {
			for _, path := range []string{"thumbnailUrl", "image.url", "image"} {
				v := strings.TrimSpace(node.Get(path).String())
				if v != "" {
					thumb = v
					return
				}
			}
			node.ForEach(func(_, child gjson.Result) bool {
				visit(child)
				return thumb == ""
			})
			return
		}
		if node.IsArray() {
			for _, child := range node.Array() {
				visit(child)
				if thumb != "" {
					return
				}
			}
		}
	}

	visit(root)
	return strings.TrimSpace(thumb)
}

func parseJSONLDChannel(data string) string {
	root := gjson.Parse(data)
	channel := ""

	var visit func(node gjson.Result)
	visit = func(node gjson.Result) {
		if channel != "" || !node.Exists() {
			return
		}
		if node.IsObject() {
			channel = cleanPMVHavenChannel(firstNonEmpty(
				strings.TrimSpace(node.Get("author.name").String()),
				strings.TrimSpace(node.Get("author").String()),
				strings.TrimSpace(node.Get("creator.name").String()),
				strings.TrimSpace(node.Get("creator").String()),
			))
			if channel != "" {
				return
			}
			node.ForEach(func(_, child gjson.Result) bool {
				visit(child)
				return channel == ""
			})
			return
		}
		if node.IsArray() {
			for _, child := range node.Array() {
				visit(child)
				if channel != "" {
					return
				}
			}
		}
	}

	visit(root)
	return channel
}

func parseJSONLDDescription(data string) string {
	root := gjson.Parse(data)
	description := ""

	var visit func(node gjson.Result)
	visit = func(node gjson.Result) {
		if description != "" || !node.Exists() {
			return
		}
		if node.IsObject() {
			description = cleanPMVHavenDescription(strings.TrimSpace(node.Get("description").String()))
			if description != "" {
				return
			}
			node.ForEach(func(_, child gjson.Result) bool {
				visit(child)
				return description == ""
			})
			return
		}
		if node.IsArray() {
			for _, child := range node.Array() {
				visit(child)
				if description != "" {
					return
				}
			}
		}
	}

	visit(root)
	return description
}

func extractPMVHavenChannel(sel *goquery.Selection) string {
	if sel == nil || sel.Length() == 0 {
		return ""
	}
	channel := cleanPMVHavenChannel(firstNonEmpty(
		firstText(sel,
			`a[rel="author"]`,
			`.author a`,
			`.entry-author a`,
			`.byline a`,
			`a[href*="/author/"]`,
		),
		channelFromAuthorURL(firstAttr(sel,
			`a[rel="author"][href]`,
			`.author a[href]`,
			`.entry-author a[href]`,
			`.byline a[href]`,
			`a[href*="/author/"][href]`,
		)),
	))
	return channel
}

func channelFromAuthorURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	u, err := url.Parse(absoluteURL(raw))
	if err != nil {
		return ""
	}

	p := strings.Trim(strings.TrimSpace(u.Path), "/")
	if p == "" {
		return ""
	}
	parts := strings.Split(p, "/")
	if len(parts) >= 2 {
		head := strings.ToLower(parts[0])
		if head == "author" || head == "channel" || head == "user" || head == "users" || head == "creator" {
			return cleanPMVHavenChannel(parts[len(parts)-1])
		}
	}
	return ""
}

func firstAttr(sel *goquery.Selection, selectors ...string) string {
	for _, selector := range selectors {
		n := sel.Find(selector).First()
		if n.Length() == 0 {
			continue
		}
		if strings.Contains(selector, "srcset") {
			if val, ok := n.Attr("data-srcset"); ok && strings.TrimSpace(val) != "" {
				return strings.TrimSpace(val)
			}
			if val, ok := n.Attr("srcset"); ok && strings.TrimSpace(val) != "" {
				return strings.TrimSpace(val)
			}
		}
		for _, attr := range []string{"href", "data-src", "data-lazy-src", "data-original", "src"} {
			if val, ok := n.Attr(attr); ok && strings.TrimSpace(val) != "" {
				return strings.TrimSpace(val)
			}
		}
	}
	return ""
}

func firstText(sel *goquery.Selection, selectors ...string) string {
	for _, selector := range selectors {
		n := sel.Find(selector).First()
		if n.Length() == 0 {
			continue
		}
		txt := strings.TrimSpace(n.Text())
		if txt != "" {
			return txt
		}
	}
	return ""
}

func attrVal(sel *goquery.Selection, attr string) string {
	if sel == nil || sel.Length() == 0 {
		return ""
	}
	val, ok := sel.Attr(attr)
	if !ok {
		return ""
	}
	return html.UnescapeString(strings.TrimSpace(val))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func looksLikeSceneURL(raw string) bool {
	u := canonicalSceneURL(raw)
	if u == "" {
		return false
	}
	l := strings.ToLower(u)
	if !strings.Contains(l, "pmvhaven.com") {
		return false
	}
	parsed, err := url.Parse(l)
	if err == nil {
		trimmedPath := strings.Trim(parsed.Path, "/")
		if trimmedPath == "" {
			return false
		}
	}
	blocked := []string{"/tag/", "/category/", "/author/", "/profile/", "/users/", "/page/", "/wp-content/", "/feed", "?s="}
	for _, b := range blocked {
		if strings.Contains(l, b) {
			return false
		}
	}
	return true
}

func canonicalSceneURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	abs := absoluteURL(raw)
	u, err := url.Parse(abs)
	if err != nil {
		return ""
	}
	u.Fragment = ""
	u.RawQuery = ""
	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String()
}

func absoluteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	base, _ := url.Parse(pmvHavenBaseURL)
	ref, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return base.ResolveReference(ref).String()
}

func buildCandidateID(sceneURL string) string {
	sceneURL = canonicalSceneURL(sceneURL)
	if sceneURL == "" {
		return ""
	}

	// Preferred PMVHaven ID: trailing 24-char hex after underscore in /video/{slug}_{id}
	if u, err := url.Parse(sceneURL); err == nil {
		base := strings.TrimSpace(path.Base(u.Path))
		re := regexp.MustCompile(`_([a-f0-9]{24})$`)
		if m := re.FindStringSubmatch(strings.ToLower(base)); len(m) == 2 {
			return m[1]
		}
	}

	u, err := url.Parse(sceneURL)
	if err == nil {
		base := strings.ToLower(strings.TrimSpace(path.Base(u.Path)))
		base = regexp.MustCompile(`[^a-z0-9\-_]+`).ReplaceAllString(base, "-")
		base = strings.Trim(base, "-")
		if base != "" {
			return base
		}
	}
	sum := sha1.Sum([]byte(sceneURL))
	return hex.EncodeToString(sum[:])[:12]
}

func titleFromSceneURL(sceneURL string) string {
	u, err := url.Parse(canonicalSceneURL(sceneURL))
	if err != nil {
		return ""
	}
	base := strings.TrimSpace(path.Base(u.Path))
	base = strings.TrimPrefix(base, "video/")
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	base = regexp.MustCompile(`\s+[a-f0-9]{24}$`).ReplaceAllString(base, "")
	base = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(base), " ")
	return cleanPMVHavenListingTitle(base, "")
}

func cleanPMVHavenTitle(raw string) string {
	title := strings.TrimSpace(html.UnescapeString(raw))
	if title == "" {
		return ""
	}

	for _, suffix := range []string{" | PMVHaven", " - PMVHaven"} {
		if strings.HasSuffix(title, suffix) {
			title = strings.TrimSpace(strings.TrimSuffix(title, suffix))
		}
	}
	return title
}

func cleanPMVHavenListingTitle(raw, sceneURL string) string {
	title := strings.TrimSpace(cleanPMVHavenTitle(raw))
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")
	if title == "" {
		return humanizePMVHavenSlugTitle(sceneURL)
	}

	// Profile/list cards sometimes collapse title + runtime + ratings + uploader + tags
	// into a single text blob. In that case prefer a clean title derived from the scene slug.
	if looksLikePMVHavenListingNoise(title) {
		if fallback := humanizePMVHavenSlugTitle(sceneURL); fallback != "" {
			return fallback
		}
	}
	return title
}

func looksLikePMVHavenListingNoise(title string) bool {
	lower := strings.ToLower(title)
	if strings.Contains(title, "#") || strings.Contains(title, "%") {
		return true
	}
	for _, marker := range []string{" ago", "nudity", "voice over", "vo", "views", "brainwashing"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	if regexp.MustCompile(`\b(?:fhd|qhd|uhd|4k|8k|3d)\b`).MatchString(lower) {
		return true
	}
	if regexp.MustCompile(`\b\d{1,2}:\d{2}\b`).MatchString(title) {
		return true
	}
	return false
}

func humanizePMVHavenSlugTitle(sceneURL string) string {
	u, err := url.Parse(canonicalSceneURL(sceneURL))
	if err != nil {
		return ""
	}
	base := strings.TrimSpace(path.Base(u.Path))
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	base = regexp.MustCompile(`\s+[a-f0-9]{24}$`).ReplaceAllString(base, "")
	base = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(base), " ")
	if base == "" {
		return ""
	}

	parts := strings.Fields(base)
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

func cleanPMVHavenChannel(raw string) string {
	channel := strings.TrimSpace(html.UnescapeString(raw))
	if channel == "" {
		return ""
	}
	channel = regexp.MustCompile(`(?i)^by\s+`).ReplaceAllString(channel, "")
	channel = strings.TrimPrefix(channel, "@")
	channel = strings.ReplaceAll(channel, "_", " ")
	channel = strings.ReplaceAll(channel, "-", " ")
	channel = strings.TrimSpace(channel)
	channel = regexp.MustCompile(`\s+`).ReplaceAllString(channel, " ")
	if channel == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(channel), "http://") || strings.HasPrefix(strings.ToLower(channel), "https://") {
		if fromURL := channelFromAuthorURL(channel); fromURL != "" {
			channel = fromURL
		}
	}
	switch strings.ToLower(channel) {
	case "pmvhaven", "author", "channel", "user", "creator":
		return ""
	}
	return channel
}

func cleanPMVHavenDescription(raw string) string {
	desc := strings.TrimSpace(raw)
	if desc == "" {
		return ""
	}
	desc = html.UnescapeString(desc)
	desc = strings.ReplaceAll(desc, "\r\n", "\n")
	desc = strings.ReplaceAll(desc, "\r", "\n")

	lines := strings.Split(desc, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = regexp.MustCompile(`\s+`).ReplaceAllString(line, " ")
		cleaned = append(cleaned, line)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}
