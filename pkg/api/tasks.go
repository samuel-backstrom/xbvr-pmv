package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/tasks"
)

type RequestScrapeJAVR struct {
	Scraper string `json:"s"`
	Query   string `json:"q"`
}

type RequestScrapeTPDB struct {
	ApiToken string `json:"apiToken"`
	SceneUrl string `json:"sceneUrl"`
}

type RequestPythonDancerBatch struct {
	Limit           int    `json:"limit"`
	Concurrency     int    `json:"concurrency"`
	VolumeID        uint   `json:"volume_id"`
	PathPrefix      string `json:"path_prefix"`
	FileID          uint   `json:"file_id"`
	ForceRegenerate bool   `json:"force_regenerate"`
	PostProcessMode string `json:"post_process_mode"`
}

type RequestSingleScrape struct {
	Site           string                            `json:"site"`
	SceneUrl       string                            `json:"sceneurl"`
	AdditionalInfo []RequestSingleScrapeAdditionInfo `json:"additionalinfo"`
}

type RequestSingleScrapeAdditionInfo struct {
	FieldName   string `json:"fieldName"`
	FieldPrompt string `json:"fieldPrompt"`
	Placeholder string `json:"placeholder"`
	FieldValue  string `json:"fieldValue"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
}

type RequestPMVMatch struct {
	FileID uint `json:"file_id"`
	DryRun bool `json:"dry_run"`
}

type RequestPMVMatchBatch struct {
	DryRun            bool   `json:"dry_run"`
	Limit             int    `json:"limit"`
	Concurrency       int    `json:"concurrency"`
	VolumeID          uint   `json:"volume_id"`
	PathPrefix        string `json:"path_prefix"`
	RefreshExisting   bool   `json:"refresh_existing"`
	UpdateTitle       bool   `json:"update_title"`
	UpdateStudio      bool   `json:"update_studio"`
	UpdateSceneURL    bool   `json:"update_scene_url"`
	UpdateThumbnail   bool   `json:"update_thumbnail"`
	UpdateDescription bool   `json:"update_description"`
}

type RequestPMVImport struct {
	URL         string `json:"url"`
	ListURL     string `json:"list_url"`
	PathPrefix  string `json:"path_prefix"`
	Limit       int    `json:"limit"`
	Concurrency int    `json:"concurrency"`
}

type ResponseBackupBundle struct {
	Response string `json:"status"`
}

type ResponseSceneScrape struct {
	Response string       `json:"status"`
	Scene    models.Scene `json:"scene"`
}

type TaskResource struct{}

func taskRequestFields(req *restful.Request, fields logrus.Fields) logrus.Fields {
	out := logrus.Fields{
		"endpoint": req.Request.URL.Path,
		"method":   req.Request.Method,
	}
	for key, value := range fields {
		out[key] = value
	}
	return out
}

func startAPITask(req *restful.Request, resp *restful.Response, task string, fields logrus.Fields, fn func()) {
	common.StartAsyncTask(task, "api", taskRequestFields(req, fields), fn)
	resp.WriteHeader(http.StatusAccepted)
}

func (i TaskResource) WebService() *restful.WebService {
	tags := []string{"Task"}

	ws := new(restful.WebService)

	ws.Path("/api/task").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/rescan").To(i.rescan).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/rescan/{storage-id}").To(i.rescan).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/scene-refresh").To(i.sceneRrefresh).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/clean-tags").To(i.cleanTags).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/scrape").To(i.scrape).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/singlescrape").To(i.singleScrape).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseSceneScrape{}))

	ws.Route(ws.GET("/index").To(i.index).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/preview/generate").To(i.previewGenerate).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/funscript/export-all").To(i.exportAllFunscripts).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/funscript/export-new").To(i.exportNewFunscripts).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/funscript/python-dancer").To(i.pythonDancerFunscripts).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(tasks.PythonDancerBatchResult{}))

	ws.Route(ws.GET("/funscript/python-dancer").To(i.pythonDancerFunscriptsTask).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/bundle/backup").To(i.backupBundle).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseBackupBundle{}))

	ws.Route(ws.POST("/bundle/restore").To(i.restoreBundle).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/scrape-javr").To(i.scrapeJAVR).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/scrape-tpdb").To(i.scrapeTPDB).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/relink_alt_aource_scenes").To(i.relink_alt_aource_scenes).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/pmv-match").To(i.pmvMatch).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(tasks.PMVMatchResult{}))

	ws.Route(ws.POST("/pmv-match-unmatched").To(i.pmvMatchUnmatched).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(tasks.PMVMatchBatchResult{}))

	ws.Route(ws.GET("/pmv-match-unmatched").To(i.pmvMatchUnmatchedTask).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/pmv-import").To(i.pmvImport).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(tasks.PMVImportResult{}))

	ws.Route(ws.POST("/pmv-import-list").To(i.pmvImportList).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(tasks.PMVImportBatchResult{}))

	return ws
}

func (i TaskResource) rescan(req *restful.Request, resp *restful.Response) {
	id, err := strconv.Atoi(req.PathParameter("storage-id"))
	if err != nil {
		startAPITask(req, resp, "rescan", logrus.Fields{
			"storage_id": "all",
		}, func() {
			tasks.RescanVolumes(-1)
		})
		return
	}

	startAPITask(req, resp, "rescan", logrus.Fields{
		"storage_id": id,
	}, func() {
		tasks.RescanVolumes(id)
	})
}

func (i TaskResource) sceneRrefresh(req *restful.Request, resp *restful.Response) {
	startAPITask(req, resp, "scene-refresh", nil, func() {
		tasks.RefreshSceneStatuses()
	})
}

func (i TaskResource) cleanTags(req *restful.Request, resp *restful.Response) {
	startAPITask(req, resp, "clean-tags", nil, func() {
		tasks.CleanTags()
	})
}

func (i TaskResource) index(req *restful.Request, resp *restful.Response) {
	startAPITask(req, resp, "search-index", nil, func() {
		tasks.SearchIndex()
	})
}

func (i TaskResource) scrape(req *restful.Request, resp *restful.Response) {
	qSiteID := req.QueryParameter("site")
	if qSiteID == "" {
		qSiteID = "_enabled"
	}
	startAPITask(req, resp, "scrape", logrus.Fields{
		"site": qSiteID,
	}, func() {
		tasks.Scrape(qSiteID, "", "")
	})
}
func (i TaskResource) singleScrape(req *restful.Request, resp *restful.Response) {
	var scrapeParams RequestSingleScrape
	req.ReadEntity(&scrapeParams)
	additionalInfo, _ := json.Marshal(scrapeParams.AdditionalInfo)

	newScene := tasks.ScrapeSingleScene(scrapeParams.Site, scrapeParams.SceneUrl, string(additionalInfo))

	createResp := &ResponseSceneScrape{
		Response: "OK",
		Scene:    newScene,
	}
	resp.WriteHeaderAndEntity(http.StatusOK, createResp)
}

func (i TaskResource) exportAllFunscripts(req *restful.Request, resp *restful.Response) {
	tasks.ExportFunscripts(resp.ResponseWriter, false)
}

func (i TaskResource) exportNewFunscripts(req *restful.Request, resp *restful.Response) {
	tasks.ExportFunscripts(resp.ResponseWriter, true)
}

func (i TaskResource) pythonDancerFunscripts(req *restful.Request, resp *restful.Response) {
	var r RequestPythonDancerBatch
	if err := req.ReadEntity(&r); err != nil {
		APIError(req, resp, http.StatusBadRequest, err)
		return
	}

	result, statusCode, err := tasks.GeneratePythonDancerFunscripts(tasks.PythonDancerBatchRequest{
		Limit:           r.Limit,
		Concurrency:     r.Concurrency,
		VolumeID:        r.VolumeID,
		PathPrefix:      r.PathPrefix,
		FileID:          r.FileID,
		ForceRegenerate: r.ForceRegenerate,
		PostProcessMode: r.PostProcessMode,
	})
	if err != nil {
		APIError(req, resp, statusCode, err)
		return
	}
	resp.WriteHeaderAndEntity(statusCode, result)
}

func (i TaskResource) pythonDancerFunscriptsTask(req *restful.Request, resp *restful.Response) {
	limit, _ := strconv.Atoi(req.QueryParameter("limit"))
	concurrency, _ := strconv.Atoi(req.QueryParameter("concurrency"))
	volumeID64, _ := strconv.ParseUint(req.QueryParameter("volume_id"), 10, 64)
	pathPrefix := strings.TrimSpace(req.QueryParameter("path_prefix"))
	postProcessMode := strings.TrimSpace(req.QueryParameter("post_process_mode"))

	startAPITask(req, resp, "python-dancer-funscripts", logrus.Fields{
		"limit":             limit,
		"concurrency":       concurrency,
		"volume_id":         volumeID64,
		"path_prefix":       pathPrefix,
		"post_process_mode": postProcessMode,
	}, func() {
		tasks.RunPythonDancerFunscriptTask(tasks.PythonDancerBatchRequest{
			Limit:           limit,
			Concurrency:     concurrency,
			VolumeID:        uint(volumeID64),
			PathPrefix:      pathPrefix,
			PostProcessMode: postProcessMode,
		})
	})
}

func (i TaskResource) pmvImport(req *restful.Request, resp *restful.Response) {
	var r RequestPMVImport
	if err := req.ReadEntity(&r); err != nil {
		APIError(req, resp, http.StatusBadRequest, err)
		return
	}

	result, statusCode, err := tasks.ImportPMVHavenVideo(tasks.PMVImportRequest{
		URL:        r.URL,
		PathPrefix: strings.TrimSpace(r.PathPrefix),
	})
	if err != nil {
		APIError(req, resp, statusCode, err)
		return
	}
	resp.WriteHeaderAndEntity(statusCode, result)
}

func (i TaskResource) pmvImportList(req *restful.Request, resp *restful.Response) {
	var r RequestPMVImport
	if err := req.ReadEntity(&r); err != nil {
		APIError(req, resp, http.StatusBadRequest, err)
		return
	}

	result, statusCode, err := tasks.ImportPMVHavenList(tasks.PMVImportRequest{
		ListURL:     r.ListURL,
		PathPrefix:  strings.TrimSpace(r.PathPrefix),
		Limit:       r.Limit,
		Concurrency: r.Concurrency,
	})
	if err != nil {
		APIError(req, resp, statusCode, err)
		return
	}
	resp.WriteHeaderAndEntity(statusCode, result)
}

func (i TaskResource) backupBundle(req *restful.Request, resp *restful.Response) {
	inclAllSites, _ := strconv.ParseBool(req.QueryParameter("allSites"))
	onlyIncludeOfficalSites, _ := strconv.ParseBool(req.QueryParameter("onlyIncludeOfficalSites"))
	inclScenes, _ := strconv.ParseBool(req.QueryParameter("inclScenes"))
	inclFileLinks, _ := strconv.ParseBool(req.QueryParameter("inclLinks"))
	inclCuepoints, _ := strconv.ParseBool(req.QueryParameter("inclCuepoints"))
	inclHistory, _ := strconv.ParseBool(req.QueryParameter("inclHistory"))
	inclPlaylists, _ := strconv.ParseBool(req.QueryParameter("inclPlaylists"))
	inclActorAkas, _ := strconv.ParseBool(req.QueryParameter("inclActorAkas"))
	inclTagGroups, _ := strconv.ParseBool(req.QueryParameter("inclTagGroups"))
	inclVolumes, _ := strconv.ParseBool(req.QueryParameter("inclVolumes"))
	inclSites, _ := strconv.ParseBool(req.QueryParameter("inclSites"))
	inclActions, _ := strconv.ParseBool(req.QueryParameter("inclActions"))
	inclExtRefs, _ := strconv.ParseBool(req.QueryParameter("inclExtRefs"))
	inclActors, _ := strconv.ParseBool(req.QueryParameter("inclActors"))
	inclActorActions, _ := strconv.ParseBool(req.QueryParameter("inclActorActions"))
	inclConfig, _ := strconv.ParseBool(req.QueryParameter("inclConfig"))
	extRefSubset := req.QueryParameter("extRefSubset")
	playlistId := req.QueryParameter("playlistId")
	download := req.QueryParameter("download")

	bundle := tasks.BackupBundle(inclAllSites, onlyIncludeOfficalSites, inclScenes, inclFileLinks, inclCuepoints, inclHistory, inclPlaylists,
		inclActorAkas, inclTagGroups, inclVolumes, inclSites, inclActions, inclExtRefs, inclActors, inclActorActions, inclConfig, extRefSubset, playlistId, "", "")
	if download == "true" {
		resp.WriteHeaderAndEntity(http.StatusOK, ResponseBackupBundle{Response: "Ready to Download from http://xxx.xxx.xxx.xxx:9999/download/xbvr-content-bundle.json"})
	} else {
		// not downloading, display the bundle data
		resp.WriteHeaderAndEntity(http.StatusOK, (bundle))
	}

}

func (i TaskResource) restoreBundle(req *restful.Request, resp *restful.Response) {
	var r tasks.RequestRestore

	if err := req.ReadEntity(&r); err != nil {
		APIError(req, resp, http.StatusInternalServerError, err)
		return
	}

	startAPITask(req, resp, "restore-bundle", logrus.Fields{
		"bundle_url": strings.TrimSpace(r.BundleUrl),
	}, func() {
		tasks.RestoreBundle(r)
	})
}

func (i TaskResource) previewGenerate(req *restful.Request, resp *restful.Response) {
	startAPITask(req, resp, "preview-generate", nil, func() {
		tasks.GeneratePreviews(nil)
	})
}

func (i TaskResource) scrapeJAVR(req *restful.Request, resp *restful.Response) {
	var r RequestScrapeJAVR
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	if r.Query != "" {
		startAPITask(req, resp, "scrape-javr", logrus.Fields{
			"scraper": r.Scraper,
			"query":   strings.TrimSpace(r.Query),
		}, func() {
			tasks.ScrapeJAVR(r.Query, r.Scraper)
		})
		return
	}

	resp.WriteHeader(http.StatusBadRequest)
}

func (i TaskResource) scrapeTPDB(req *restful.Request, resp *restful.Response) {
	var r RequestScrapeTPDB
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	if r.ApiToken != "" && r.SceneUrl != "" {
		startAPITask(req, resp, "scrape-tpdb", logrus.Fields{
			"scene_url": strings.TrimSpace(r.SceneUrl),
		}, func() {
			tasks.ScrapeTPDB(strings.TrimSpace(r.ApiToken), strings.TrimSpace(r.SceneUrl))
		})
		return
	}

	resp.WriteHeader(http.StatusBadRequest)
}
func (i TaskResource) relink_alt_aource_scenes(req *restful.Request, resp *restful.Response) {
	startAPITask(req, resp, "relink-alt-source-scenes", nil, func() {
		tasks.MatchAlternateSources()
	})
}

func (i TaskResource) pmvMatch(req *restful.Request, resp *restful.Response) {
	var r RequestPMVMatch
	if err := req.ReadEntity(&r); err != nil {
		APIError(req, resp, http.StatusBadRequest, err)
		return
	}

	result, statusCode, err := tasks.MatchPMVFile(r.FileID, r.DryRun)
	if err != nil {
		APIError(req, resp, statusCode, err)
		return
	}
	resp.WriteHeaderAndEntity(statusCode, result)
}

func (i TaskResource) pmvMatchUnmatched(req *restful.Request, resp *restful.Response) {
	var r RequestPMVMatchBatch
	if err := req.ReadEntity(&r); err != nil {
		APIError(req, resp, http.StatusBadRequest, err)
		return
	}

	result, statusCode, err := tasks.MatchPMVUnmatchedFiles(tasks.PMVMatchBatchRequest{
		DryRun:            r.DryRun,
		Limit:             r.Limit,
		Concurrency:       r.Concurrency,
		VolumeID:          r.VolumeID,
		PathPrefix:        r.PathPrefix,
		RefreshExisting:   r.RefreshExisting,
		UpdateTitle:       r.UpdateTitle,
		UpdateStudio:      r.UpdateStudio,
		UpdateSceneURL:    r.UpdateSceneURL,
		UpdateThumbnail:   r.UpdateThumbnail,
		UpdateDescription: r.UpdateDescription,
	})
	if err != nil {
		APIError(req, resp, statusCode, err)
		return
	}
	resp.WriteHeaderAndEntity(statusCode, result)
}

func (i TaskResource) pmvMatchUnmatchedTask(req *restful.Request, resp *restful.Response) {
	limit, _ := strconv.Atoi(req.QueryParameter("limit"))
	concurrency, _ := strconv.Atoi(req.QueryParameter("concurrency"))
	dryRun, _ := strconv.ParseBool(req.QueryParameter("dry_run"))
	refreshExisting, _ := strconv.ParseBool(req.QueryParameter("refresh_existing"))
	updateTitle, _ := strconv.ParseBool(req.QueryParameter("update_title"))
	updateStudio, _ := strconv.ParseBool(req.QueryParameter("update_studio"))
	updateSceneURL, _ := strconv.ParseBool(req.QueryParameter("update_scene_url"))
	updateThumbnail, _ := strconv.ParseBool(req.QueryParameter("update_thumbnail"))
	updateDescription, _ := strconv.ParseBool(req.QueryParameter("update_description"))
	volumeID64, _ := strconv.ParseUint(req.QueryParameter("volume_id"), 10, 64)
	pathPrefix := strings.TrimSpace(req.QueryParameter("path_prefix"))

	startAPITask(req, resp, "pmv-match-unmatched", logrus.Fields{
		"dry_run":            dryRun,
		"limit":              limit,
		"concurrency":        concurrency,
		"volume_id":          volumeID64,
		"path_prefix":        pathPrefix,
		"refresh_existing":   refreshExisting,
		"update_title":       updateTitle,
		"update_studio":      updateStudio,
		"update_scene_url":   updateSceneURL,
		"update_thumbnail":   updateThumbnail,
		"update_description": updateDescription,
	}, func() {
		tasks.RunPMVMatchUnmatchedTask(tasks.PMVMatchBatchRequest{
			DryRun:            dryRun,
			Limit:             limit,
			Concurrency:       concurrency,
			VolumeID:          uint(volumeID64),
			PathPrefix:        pathPrefix,
			RefreshExisting:   refreshExisting,
			UpdateTitle:       updateTitle,
			UpdateStudio:      updateStudio,
			UpdateSceneURL:    updateSceneURL,
			UpdateThumbnail:   updateThumbnail,
			UpdateDescription: updateDescription,
		})
	})
}
