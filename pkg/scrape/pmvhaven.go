package scrape

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"html"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
		limit = 5
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
		if c.SceneURL == "" || c.Title == "" || seen[c.SceneURL] {
			return false
		}
		if c.ID == "" {
			c.ID = buildCandidateID(c.SceneURL)
		}
		seen[c.SceneURL] = true
		out = append(out, c)
		return len(out) >= limit
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

		title := firstText(card,
			`.entry-title`,
			`h1`,
			`h2`,
			`h3`,
			`a[rel="bookmark"]`,
			`a`,
		)

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

		title := strings.TrimSpace(firstNonEmpty(
			attrVal(a, "title"),
			attrVal(a, "aria-label"),
			strings.TrimSpace(a.Text()),
		))

		img := a.Find("img").First()
		if img.Length() == 0 {
			// Fallback: some layouts place image in adjacent wrappers.
			img = a.Parent().Find("img").First()
		}

		if title == "" && img.Length() > 0 {
			title = strings.TrimSpace(firstNonEmpty(
				attrVal(img, "alt"),
				attrVal(img, "title"),
			))
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

	return out
}

func parseJSONLDCandidates(data string) []PMVHavenCandidate {
	out := []PMVHavenCandidate{}
	seen := map[string]bool{}

	appendCandidate := func(title, sceneURL, thumbnailURL string) {
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
		appendCandidate(title, sceneURL, thumbnailURL)
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
	blocked := []string{"/tag/", "/category/", "/author/", "/page/", "/wp-content/", "/feed", "?s="}
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
	return base
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
