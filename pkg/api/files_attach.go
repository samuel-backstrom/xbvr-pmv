package api

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/models"
)

type RequestAttachFunscript struct {
	SceneID      uint `json:"scene_id"`
	VideoFileID  uint `json:"video_file_id"`
	ScriptFileID uint `json:"script_file_id"`
}

func decodeSceneFilenames(raw string) []string {
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}

func encodeSceneFilenames(values []string) string {
	payload, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func appendUniqueFilename(values []string, filename string) []string {
	for _, value := range values {
		if value == filename {
			return values
		}
	}
	return append(values, filename)
}

func removeFilename(values []string, filename string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != filename {
			out = append(out, value)
		}
	}
	return out
}

func (i FilesResource) attachFunscript(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var r RequestAttachFunscript
	if err := req.ReadEntity(&r); err != nil {
		log.Error(err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if r.SceneID == 0 || r.VideoFileID == 0 || r.ScriptFileID == 0 {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	var scene models.Scene
	if err := scene.GetIfExistByPK(r.SceneID); err != nil {
		log.WithError(err).Warn("failed to load scene for funscript attachment")
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	var videoFile models.File
	if err := db.Preload("Volume").Where(&models.File{ID: r.VideoFileID}).First(&videoFile).Error; err != nil {
		log.WithError(err).Warn("failed to load target video file for funscript attachment")
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	var scriptFile models.File
	if err := db.Preload("Volume").Where(&models.File{ID: r.ScriptFileID}).First(&scriptFile).Error; err != nil {
		log.WithError(err).Warn("failed to load script file for funscript attachment")
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	if videoFile.Type != "video" || scriptFile.Type != "script" {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if scriptFile.Volume.Type != "local" {
		log.WithField("volume_type", scriptFile.Volume.Type).Warn("funscript attachment only supports local files")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	targetFilename := strings.TrimSuffix(videoFile.Filename, filepath.Ext(videoFile.Filename)) + ".funscript"
	oldFilename := scriptFile.Filename
	targetPath := filepath.Join(scriptFile.Path, targetFilename)
	currentPath := scriptFile.GetPath()

	if targetFilename != oldFilename {
		if _, err := os.Stat(targetPath); err == nil && targetPath != currentPath {
			log.WithField("target", targetPath).Warn("funscript attachment target already exists")
			resp.WriteHeader(http.StatusConflict)
			return
		} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.WithError(err).Warn("failed to inspect funscript attachment target")
			resp.WriteHeader(http.StatusBadGateway)
			return
		}

		if err := os.Rename(currentPath, targetPath); err != nil {
			log.WithError(err).Warn("failed to rename funscript for attachment")
			resp.WriteHeader(http.StatusBadGateway)
			return
		}

		scriptFile.Filename = targetFilename
	}

	previousSceneID := scriptFile.SceneID
	scriptFile.SceneID = scene.ID
	if err := db.Save(&scriptFile).Error; err != nil {
		log.WithError(err).Warn("failed to save funscript attachment")
		resp.WriteHeader(http.StatusBadGateway)
		return
	}

	sceneFilenames := decodeSceneFilenames(scene.FilenamesArr)
	sceneFilenames = removeFilename(sceneFilenames, oldFilename)
	sceneFilenames = appendUniqueFilename(sceneFilenames, targetFilename)
	scene.FilenamesArr = encodeSceneFilenames(sceneFilenames)
	if err := db.Save(&scene).Error; err != nil {
		log.WithError(err).Warn("failed to save scene after funscript attachment")
		resp.WriteHeader(http.StatusBadGateway)
		return
	}

	if previousSceneID != 0 && previousSceneID != scene.ID {
		var previousScene models.Scene
		if err := previousScene.GetIfExistByPK(previousSceneID); err == nil {
			previousScene.FilenamesArr = encodeSceneFilenames(removeFilename(decodeSceneFilenames(previousScene.FilenamesArr), oldFilename))
			if err := db.Save(&previousScene).Error; err == nil {
				previousScene.UpdateStatus()
			}
		}
	}

	models.AddAction(scene.SceneID, "match", "filenames_arr", scene.FilenamesArr)
	scene.UpdateStatus()

	if err := scene.GetIfExistByPK(scene.ID); err != nil {
		log.WithError(err).Warn("failed to reload scene after funscript attachment")
		resp.WriteHeader(http.StatusBadGateway)
		return
	}

	log.WithFields(logrus.Fields{
		"scene_id":       scene.SceneID,
		"video_file_id":  videoFile.ID,
		"script_file_id": scriptFile.ID,
		"target_name":    targetFilename,
	}).Info("Attached funscript to scene")

	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}
