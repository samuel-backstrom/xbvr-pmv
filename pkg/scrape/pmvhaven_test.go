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

func TestParsePMVHavenSearchHTML_ExtractsChannelFromAuthor(t *testing.T) {
	html := `
	<html><body>
	  <article class="post">
	    <h2 class="entry-title"><a href="/video/some-title_673a8cccaa8d005d3a4d0ae8/">Some Title</a></h2>
	    <div class="byline"><a href="/author/adhdpmv/">by ADHDPMV</a></div>
	  </article>
	</body></html>`

	candidates := ParsePMVHavenSearchHTML(html, 5)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Channel != "ADHDPMV" {
		t.Fatalf("unexpected channel %q", candidates[0].Channel)
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

func TestParsePMVHavenSceneHTMLForChannel_AuthorLink(t *testing.T) {
	html := `
	<html><body>
	  <div class="entry-meta">
	    <a rel="author" href="/author/account-for-combustion/">By Account For Combustion</a>
	  </div>
	</body></html>`

	got := ParsePMVHavenSceneHTMLForChannel(html)
	want := "Account For Combustion"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParsePMVHavenSceneHTMLForChannel_JSONLD(t *testing.T) {
	html := `
	<html><body>
	  <script type="application/ld+json">
	  {
	    "@context": "https://schema.org",
	    "@type": "VideoObject",
	    "name": "Some PMV",
	    "author": { "@type": "Person", "name": "ADHDPMV" }
	  }
	  </script>
	</body></html>`

	got := ParsePMVHavenSceneHTMLForChannel(html)
	want := "ADHDPMV"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParsePMVHavenSceneHTMLForMediaURL(t *testing.T) {
	html := `
	<html><body>
	  <script>
	    window.__NUXT__ = {
	      data: [{
	        video: {
	          src: "https://video.pmvhaven.com/videos/Interceptor_-_Tit-Tacular_2_1772909166965_qqczh57k.mp4",
	          hls: "https://video.pmvhaven.com/videos/Interceptor_-_Tit-Tacular_2_1772909166965_qqczh57k.mp4/master.m3u8"
	        }
	      }]
	    }
	  </script>
	</body></html>`

	got := ParsePMVHavenSceneHTMLForMediaURL(html)
	want := "https://video.pmvhaven.com/videos/Interceptor_-_Tit-Tacular_2_1772909166965_qqczh57k.mp4"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParsePMVHavenSceneHTMLForMediaURL_PrefersDirectMP4OverHLS(t *testing.T) {
	html := `
	<html><body>
	  <script type="application/ld+json">
	  {
	    "@context": "https://schema.org",
	    "@type": "VideoObject",
	    "contentUrl": "https://video.pmvhaven.com/videos/1772497845951_95tp2e1hqpv_0302_55_5_RIGHT_ONLY_auto_subject_v1.8.6_LRF_Full_SBS.mp4/master.m3u8"
	  }
	  </script>
	  <script>
	    window.__NUXT__={"video":{"videoUrl":"https:\\u002F\\u002Fvideo.pmvhaven.com\\u002Fvideos\\u002FFuckyou2wice_-_3D_Gooner_Porn_-_4K60_3D_Full_SBS_-_Gooning_Encouragement_and_JOI_1772498453103_4q0alo1v.mp4"}}
	  </script>
	</body></html>`

	got := ParsePMVHavenSceneHTMLForMediaURL(html)
	want := "https://video.pmvhaven.com/videos/1772497845951_95tp2e1hqpv_0302_55_5_RIGHT_ONLY_auto_subject_v1.8.6_LRF_Full_SBS.mp4"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParsePMVHavenSceneHTMLForMediaURL_PrefersCurrentVideoObjectOverRelatedVideos(t *testing.T) {
	html := `
	<html><head>
	  <meta property="og:title" content="The Ten Commandments of Gooning | Goonology - A Gooning Religion | Gooning Encouragement and JOI - PMVHaven" />
	  <link rel="canonical" href="https://pmvhaven.com/video/the-ten-commandments-of-gooning-goonology-a-gooning-religion-gooning-encourageme_6877c7cce11b6fa8d4b06aff" />
	</head><body>
	  <script type="application/ld+json">
	  {
	    "@context": "https://schema.org",
	    "@type": "VideoObject",
	    "name": "The Ten Commandments of Gooning | Goonology - A Gooning Religion | Gooning Encouragement and JOI",
	    "url": "https://pmvhaven.com/video/the-ten-commandments-of-gooning-goonology-a-gooning-religion-gooning-encourageme_6877c7cce11b6fa8d4b06aff",
	    "contentUrl": "https://video.pmvhaven.com/videos/Fuckyou2wice_-_The_Ten_Commandments_of_Gooning_Goonology_-_A_Gooning_Religion_Gooning_Encouragement_and_JOI.mp4/master.m3u8"
	  }
	  </script>
	  <script>
	    window.__NUXT__={"recommended":[{"videoUrl":"https:\\u002F\\u002Fvideo.pmvhaven.com\\u002Fvideos\\u002FChickeNuggs_-_Ass_Shaking_1768324946359_yq0fib65.mp4"}]}
	  </script>
	</body></html>`

	got := ParsePMVHavenSceneHTMLForMediaURL(html)
	want := "https://video.pmvhaven.com/videos/Fuckyou2wice_-_The_Ten_Commandments_of_Gooning_Goonology_-_A_Gooning_Religion_Gooning_Encouragement_and_JOI.mp4"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParsePMVHavenSceneHTMLForMediaURL_UsesCurrentNuxtVideoURLWhenJSONLDOnlyHasKeyedHLS(t *testing.T) {
	html := `
	<html><head>
	  <meta property="og:title" content="Porn Made Me a Gooner | Gooning Encouragement and JOI - PMVHaven" />
	</head><body>
	  <script type="application/ld+json">
	  {
	    "@context": "https://schema.org",
	    "@type": "VideoObject",
	    "name": "Porn Made Me a Gooner | Gooning Encouragement and JOI",
	    "contentUrl": "https://video.pmvhaven.com/6903a6dbfb452a766a293994/master.m3u8"
	  }
	  </script>
	  <script type="application/json" data-nuxt-data="nuxt-app" data-ssr="true" id="__NUXT_DATA__">
	  [{"video":1},{"title":2,"videoUrl":3},"Porn Made Me a Gooner | Gooning Encouragement and JOI","https://video.pmvhaven.com/6903a6dbfb452a766a293994/Fuckyou2wice_-_Porn_Made_Me_a_Gooner_Gooning_Encouragement_and_JOI.mp4"]
	  </script>
	</body></html>`

	got := ParsePMVHavenSceneHTMLForMediaURL(html)
	want := "https://video.pmvhaven.com/6903a6dbfb452a766a293994/Fuckyou2wice_-_Porn_Made_Me_a_Gooner_Gooning_Encouragement_and_JOI.mp4"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParsePMVHavenSceneHTMLForDescription_MetaAndJSON(t *testing.T) {
	html := `
	<html><head>
	  <meta name="description" content="Short summary" />
	  <meta property="og:description" content="Short summary" />
	</head><body>
	  <script>
	    window.__NUXT__={"description":"Short summary\nFull description line 2"}
	  </script>
	</body></html>`

	got := ParsePMVHavenSceneHTMLForDescription(html)
	want := "Short summary\nFull description line 2"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParsePMVHavenSearchHTML_ProfileCardFallsBackToSlugTitle(t *testing.T) {
	html := `
	<html><body>
	  <a href="/video/obsessive-porn-disorder-gooning-encouragement-and-joi-4k60_696597d389abf235255bcef6">
	    4K16:910:27Obsessive Porn Disorder | Gooning Encouragement and JOI | 4K6095%FFuckyou2wice115.9K2mo ago #Gooning encouragement #Brainwashing+3Aim To Head - Black DiamondNudityVO
	  </a>
	</body></html>`

	candidates := ParsePMVHavenSearchHTML(html, 5)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	want := "Obsessive Porn Disorder Gooning Encouragement And Joi 4k60"
	if candidates[0].Title != want {
		t.Fatalf("expected title %q, got %q", want, candidates[0].Title)
	}
}

func TestParsePMVHavenSearchHTML_ProfileGridUsesVisibleCardMetadata(t *testing.T) {
	html := `
	<html>
	  <head>
	    <title>Fuckyou2wice's Profile - PMVHaven</title>
	    <meta property="og:type" content="profile" />
	    <meta name="description" content="Fuckyou2wice on PMVHaven - 53 videos uploaded." />
	  </head>
	  <body>
	    <div class="videos-grid-fixed">
	      <a href="/video/3d-gooner-porn-4k60-3d-full-sbs-gooning-encouragement-and-joi_69a62e15f549d9461c462f2e?from=user-profile" data-video-id="69a62e15f549d9461c462f2e">
	        <img src="https://video.pmvhaven.com/thumbnails/69a62e15f549d9461c462f2e/thumb_lg.webp" alt="3D Gooner Porn - 4K60 3D Full SBS - Gooning Encouragement and JOI" />
	        <h3 class="line-clamp-2">3D Gooner Porn - 4K60 3D Full SBS - Gooning Encouragement and JOI</h3>
	        <span role="link"><span class="truncate">Fuckyou2wice</span></span>
	      </a>
	    </div>
	  </body>
	</html>`

	candidates := ParsePMVHavenSearchHTML(html, 5)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Title != "3D Gooner Porn - 4K60 3D Full SBS - Gooning Encouragement and JOI" {
		t.Fatalf("unexpected title %q", candidates[0].Title)
	}
	if candidates[0].SceneURL != "https://pmvhaven.com/video/3d-gooner-porn-4k60-3d-full-sbs-gooning-encouragement-and-joi_69a62e15f549d9461c462f2e" {
		t.Fatalf("unexpected scene url %q", candidates[0].SceneURL)
	}
	if candidates[0].ThumbnailURL != "https://video.pmvhaven.com/thumbnails/69a62e15f549d9461c462f2e/thumb_lg.webp" {
		t.Fatalf("unexpected thumbnail %q", candidates[0].ThumbnailURL)
	}
	if candidates[0].Channel != "Fuckyou2wice" {
		t.Fatalf("unexpected channel %q", candidates[0].Channel)
	}
}

func TestParsePMVHavenSearchHTML_ProfileNuxtPayloadAddsHiddenVideos(t *testing.T) {
	html := `
	<html>
	  <head>
	    <title>Fuckyou2wice's Profile - PMVHaven</title>
	    <meta property="og:type" content="profile" />
	  </head>
	  <body>
	    <div class="videos-grid-fixed">
	      <a href="/video/visible-video_69a62e15f549d9461c462f2e?from=user-profile" data-video-id="69a62e15f549d9461c462f2e">
	        <img src="https://video.pmvhaven.com/thumbnails/69a62e15f549d9461c462f2e/thumb_lg.webp" alt="Visible Video" />
	        <h3 class="line-clamp-2">Visible Video</h3>
	        <span role="link"><span class="truncate">Fuckyou2wice</span></span>
	      </a>
	    </div>
	    <script type="application/json" data-nuxt-data="nuxt-app" data-ssr="true" id="__NUXT_DATA__">[
	      ["ShallowReactive",1],
	      {"data":2},
	      ["ShallowReactive",3],
	      {"user-profile-Fuckyou2wice":4},
	      {"_id":5,"title":6,"uploaderUsername":7,"videoUrl":8,"thumbnailUrl":9},
	      "69a62e15f549d9461c462f30",
	      "Hidden Payload Video",
	      "Fuckyou2wice",
	      "https:\\u002F\\u002Fvideo.pmvhaven.com\\u002Fvideos\\u002FFuckyou2wice_-_Hidden_Payload_Video.mp4",
	      "https:\\u002F\\u002Fvideo.pmvhaven.com\\u002Fthumbnails\\u002F69a62e15f549d9461c462f30\\u002Fthumb_lg.webp"
	    ]</script>
	  </body>
	</html>`

	candidates := ParsePMVHavenSearchHTML(html, 20)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[1].Title != "Hidden Payload Video" {
		t.Fatalf("unexpected payload title %q", candidates[1].Title)
	}
	if candidates[1].SceneURL != "https://pmvhaven.com/video/hidden-payload-video_69a62e15f549d9461c462f30" {
		t.Fatalf("unexpected payload scene url %q", candidates[1].SceneURL)
	}
	if candidates[1].Channel != "Fuckyou2wice" {
		t.Fatalf("unexpected payload channel %q", candidates[1].Channel)
	}
}
