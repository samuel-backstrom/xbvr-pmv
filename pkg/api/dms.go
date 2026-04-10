package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/session"
)

type DMSResource struct{}

func (i DMSResource) WebService() *restful.WebService {
	tags := []string{"DMS"}

	ws := new(restful.WebService)

	ws.Path("/api/dms").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/file/{file-id}").To(i.getFile).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		ContentEncodingEnabled(false).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/file/{file-id}/{var:*}").To(i.getFile).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		ContentEncodingEnabled(false).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/heatmap/{file-id}").To(i.getHeatmap).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		ContentEncodingEnabled(false).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/preview/{scene-id}").To(i.getPreview).
		Param(ws.PathParameter("scene-id", "Scene ID")).
		ContentEncodingEnabled(false).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i DMSResource) getPreview(req *restful.Request, resp *restful.Response) {
	sceneID := req.PathParameter("scene-id")
	http.ServeFile(resp.ResponseWriter, req.Request, filepath.Join(common.VideoPreviewDir, fmt.Sprintf("%v.mp4", sceneID)))
}

func (i DMSResource) getHeatmap(req *restful.Request, resp *restful.Response) {
	fileID := req.PathParameter("file-id")
	http.ServeFile(resp.ResponseWriter, req.Request, filepath.Join(common.ScriptHeatmapDir, fmt.Sprintf("heatmap-%v.png", fileID)))
}

func (i DMSResource) getFile(req *restful.Request, resp *restful.Response) {
	doNotTrack := req.QueryParameter("dnt")
	id, err := strconv.Atoi(req.PathParameter("file-id"))
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if scene exist
	db, _ := models.GetDB()
	defer db.Close()

	f := models.File{}
	err = db.Preload("Volume").First(&f, id).Error
	if err != nil {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	if req.PathParameter("var") == "raw" {
		i.getRawFile(req, resp, f)
		return
	}

	switch f.Volume.Type {
	case "local":
		// Track current session
		setDeoPlayerHost(req)
		session.TrackSessionFromFile(f, doNotTrack)

		ctx := req.Request.Context()
		http.ServeFile(resp.ResponseWriter, req.Request, f.GetPath())
		select {
		case <-ctx.Done():
			session.FinishTrackingFromFile(doNotTrack)
			return
		default:
		}
	case "putio":
		id, err := strconv.ParseInt(f.Path, 10, 64)
		if err != nil {
			return
		}
		client := f.Volume.GetPutIOClient()
		url, err := client.Files.URL(context.Background(), id, false)
		if err != nil {
			return
		}
		http.Redirect(resp.ResponseWriter, req.Request, url, http.StatusFound)
	}
}

func (i DMSResource) getRawFile(req *restful.Request, resp *restful.Response, file models.File) {
	switch file.Volume.Type {
	case "local":
		http.ServeFile(resp.ResponseWriter, req.Request, file.GetPath())
	case "putio":
		id, err := strconv.ParseInt(file.Path, 10, 64)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			return
		}

		client := file.Volume.GetPutIOClient()
		url, err := client.Files.URL(req.Request.Context(), id, false)
		if err != nil {
			log.WithError(err).Warn("failed to build putio raw file url")
			resp.WriteHeader(http.StatusBadGateway)
			return
		}

		rawReq, err := http.NewRequestWithContext(req.Request.Context(), http.MethodGet, url, nil)
		if err != nil {
			log.WithError(err).Warn("failed to create putio raw file request")
			resp.WriteHeader(http.StatusBadGateway)
			return
		}

		rawResp, err := http.DefaultClient.Do(rawReq)
		if err != nil {
			log.WithError(err).Warn("failed to fetch putio raw file")
			resp.WriteHeader(http.StatusBadGateway)
			return
		}
		defer rawResp.Body.Close()

		if rawResp.StatusCode < 200 || rawResp.StatusCode >= 300 {
			log.WithField("status", rawResp.Status).Warn("putio raw file fetch returned non-success status")
			resp.WriteHeader(http.StatusBadGateway)
			return
		}

		contentType := rawResp.Header.Get("Content-Type")
		if contentType != "" {
			resp.ResponseWriter.Header().Set("Content-Type", contentType)
		}
		resp.WriteHeader(http.StatusOK)
		if _, err := io.Copy(resp.ResponseWriter, rawResp.Body); err != nil {
			log.WithError(err).Warn("failed to stream putio raw file")
		}
	default:
		resp.WriteHeader(http.StatusNotFound)
	}
}
