package scrape

import "testing"

func TestParsePMVHavenSearchHTML_ArticleCards(t *testing.T) {
	html := `
	<html><body>
	  <article class="post">
	    <h2 class="entry-title"><a href="/video-one/">Video One</a></h2>
	    <img data-src="https://cdn.pmvhaven.com/thumbs/video-one.jpg" />
	  </article>
	  <article class="post">
	    <h2 class="entry-title"><a href="https://pmvhaven.com/video-two/">Video Two</a></h2>
	    <img src="/images/video-two.jpg" />
	  </article>
	</body></html>`

	candidates := ParsePMVHavenSearchHTML(html, 5)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	if candidates[0].SceneURL != "https://pmvhaven.com/video-one" {
		t.Fatalf("unexpected scene url %q", candidates[0].SceneURL)
	}
	if candidates[0].ThumbnailURL != "https://cdn.pmvhaven.com/thumbs/video-one.jpg" {
		t.Fatalf("unexpected thumbnail %q", candidates[0].ThumbnailURL)
	}

	if candidates[1].SceneURL != "https://pmvhaven.com/video-two" {
		t.Fatalf("unexpected scene url %q", candidates[1].SceneURL)
	}
	if candidates[1].ThumbnailURL != "https://pmvhaven.com/images/video-two.jpg" {
		t.Fatalf("unexpected thumbnail %q", candidates[1].ThumbnailURL)
	}
}

func TestParsePMVHavenSearchHTML_JSONLDFallback(t *testing.T) {
	html := `
	<html><body>
	  <script type="application/ld+json">
	  {
	    "@context": "https://schema.org",
	    "@type": "VideoObject",
	    "name": "JSONLD Video",
	    "url": "https://pmvhaven.com/jsonld-video/",
	    "thumbnailUrl": "https://pmvhaven.com/thumbs/jsonld.jpg"
	  }
	  </script>
	</body></html>`

	candidates := ParsePMVHavenSearchHTML(html, 5)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Title != "JSONLD Video" {
		t.Fatalf("unexpected title %q", candidates[0].Title)
	}
	if candidates[0].SceneURL != "https://pmvhaven.com/jsonld-video" {
		t.Fatalf("unexpected scene url %q", candidates[0].SceneURL)
	}
	if candidates[0].ThumbnailURL != "https://pmvhaven.com/thumbs/jsonld.jpg" {
		t.Fatalf("unexpected thumbnail %q", candidates[0].ThumbnailURL)
	}
}

func TestParsePMVHavenSearchHTML_NuxtVideoLinks(t *testing.T) {
	html := `
	<html><body>
	  <a href="/video/gooning-is-healthy_673a8cccaa8d005d3a4d0ae8?from=search&amp;cp=0">
	    <img src="https://video.pmvhaven.com/thumbnails/673a8cccaa8d005d3a4d0ae8/thumb_lg.webp" alt="Gooning Is Healthy" />
	  </a>
	  <a href="/video/another-title_6737b7bf8d304b135bf0c4bc?from=search&amp;cp=1">
	    <img data-src="https://video.pmvhaven.com/thumbnails/6737b7bf8d304b135bf0c4bc/thumb_lg.webp" alt="Another Title" />
	  </a>
	</body></html>`

	candidates := ParsePMVHavenSearchHTML(html, 5)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	if candidates[0].ID != "673a8cccaa8d005d3a4d0ae8" {
		t.Fatalf("unexpected id %q", candidates[0].ID)
	}
	if candidates[0].SceneURL != "https://pmvhaven.com/video/gooning-is-healthy_673a8cccaa8d005d3a4d0ae8" {
		t.Fatalf("unexpected scene url %q", candidates[0].SceneURL)
	}
	if candidates[0].Title != "Gooning Is Healthy" {
		t.Fatalf("unexpected title %q", candidates[0].Title)
	}
}

func TestParsePMVHavenSceneHTMLForThumbnail_Meta(t *testing.T) {
	html := `
	<html><head>
	  <meta property="og:image" content="/images/cover.jpg" />
	  <meta name="twitter:image" content="/images/twitter.jpg" />
	</head><body></body></html>`

	got := ParsePMVHavenSceneHTMLForThumbnail(html)
	want := "https://pmvhaven.com/images/cover.jpg"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParsePMVHavenSceneHTMLForThumbnail_JSONLD(t *testing.T) {
	html := `
	<html><body>
	  <script type="application/ld+json">
	  {
	    "@context": "https://schema.org",
	    "@type": "VideoObject",
	    "thumbnailUrl": "https://cdn.pmvhaven.com/thumbs/scene.jpg"
	  }
	  </script>
	</body></html>`

	got := ParsePMVHavenSceneHTMLForThumbnail(html)
	want := "https://cdn.pmvhaven.com/thumbs/scene.jpg"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParsePMVHavenSceneHTMLForThumbnail_VideoPoster(t *testing.T) {
	html := `
	<html><body>
	  <video poster="//video.pmvhaven.com/poster.webp"></video>
	</body></html>`

	got := ParsePMVHavenSceneHTMLForThumbnail(html)
	want := "https://video.pmvhaven.com/poster.webp"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParsePMVHavenSceneHTMLForTitle_Meta(t *testing.T) {
	html := `
	<html><head>
	  <meta property="og:title" content="THROAT GOAT BLOWJOB PMV | PMVHaven" />
	</head><body></body></html>`

	got := ParsePMVHavenSceneHTMLForTitle(html)
	want := "THROAT GOAT BLOWJOB PMV"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParsePMVHavenSceneHTMLForTitle_TitleTagFallback(t *testing.T) {
	html := `
	<html><head>
	  <title>HEAVEN | PMV [Arckom] - PMVHaven</title>
	</head><body></body></html>`

	got := ParsePMVHavenSceneHTMLForTitle(html)
	want := "HEAVEN | PMV [Arckom]"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
