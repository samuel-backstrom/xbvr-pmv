package api

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/ffprobe"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/tasks"
)

type RequestGenerateThumbnail struct {
	SceneID     uint `json:"scene_id"`
	VideoFileID uint `json:"video_file_id"`
}

func decodeEditableSceneImages(raw string) []editableSceneImage {
	images := make([]editableSceneImage, 0)
	if strings.TrimSpace(raw) == "" {
		return images
	}
	if err := json.Unmarshal([]byte(raw), &images); err != nil {
		return make([]editableSceneImage, 0)
	}
	return images
}

func encodeEditableSceneImages(images []editableSceneImage) string {
	payload, err := json.Marshal(images)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func randomThumbnailSeekSeconds(duration float64, rng *rand.Rand) float64 {
	if duration <= 0 {
		return 0
	}

	start := 0.0
	end := duration
	if duration > 20 {
		start = duration * 0.10
		end = duration * 0.90
	} else if duration > 5 {
		start = 1
		end = duration - 1
	}

	if end <= start {
		return duration / 2
	}

	return start + rng.Float64()*(end-start)
}

func (i FilesResource) generateThumbnail(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var r RequestGenerateThumbnail
	if err := req.ReadEntity(&r); err != nil {
		log.WithError(err).Warn("failed to read generate-thumbnail request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if r.SceneID == 0 || r.VideoFileID == 0 {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	var scene models.Scene
	if err := scene.GetIfExistByPK(r.SceneID); err != nil {
		log.WithError(err).Warn("failed to load scene for thumbnail generation")
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	var videoFile models.File
	if err := db.Preload("Volume").Where(&models.File{ID: r.VideoFileID}).First(&videoFile).Error; err != nil {
		log.WithError(err).Warn("failed to load video file for thumbnail generation")
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	if videoFile.Type != "video" {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if videoFile.Volume.Type != "local" {
		log.WithField("volume_type", videoFile.Volume.Type).Warn("thumbnail generation only supports local video files")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	inputPath := videoFile.GetPath()
	if _, err := os.Stat(inputPath); err != nil {
		log.WithError(err).Warn("video file for thumbnail generation is missing")
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	duration := videoFile.VideoDuration
	if duration <= 0 {
		if probe, err := ffprobe.GetProbeData(inputPath, 10*time.Second); err == nil && probe != nil && probe.Format.DurationSeconds > 0 {
			duration = probe.Format.DurationSeconds
		}
	}
	if duration <= 0 {
		log.WithField("file_id", videoFile.ID).Warn("unable to determine video duration for thumbnail generation")
		resp.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	thumbDir := filepath.Join(common.MyFilesDir, "generated-thumbnails", scene.SceneID)
	if err := os.MkdirAll(thumbDir, os.ModePerm); err != nil {
		log.WithError(err).Warn("failed to create thumbnail directory")
		resp.WriteHeader(http.StatusBadGateway)
		return
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	seek := randomThumbnailSeekSeconds(duration, rng)
	fileName := fmt.Sprintf("%d-%d-%d.jpg", scene.ID, videoFile.ID, rng.Int63())
	outputPath := filepath.Join(thumbDir, fileName)
	outputURL := "/myfiles/generated-thumbnails/" + scene.SceneID + "/" + fileName

	args := []string{
		"-y",
		"-hide_banner",
		"-loglevel", "error",
		"-ss", fmt.Sprintf("%.3f", seek),
		"-i", inputPath,
		"-frames:v", "1",
		"-q:v", "2",
		"-vf", "scale=w=min(1280\\,iw):h=-2:flags=lanczos",
		outputPath,
	}

	cmd := exec.Command(tasks.GetBinPath("ffmpeg"), args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.WithFields(logrus.Fields{
			"scene_id": scene.ID,
			"file_id":  videoFile.ID,
			"seek":     seek,
			"stderr":   strings.TrimSpace(string(output)),
		}).WithError(err).Warn("failed to generate scene thumbnail")
		resp.WriteHeader(http.StatusBadGateway)
		return
	}

	images := decodeEditableSceneImages(scene.Images)
	filtered := make([]editableSceneImage, 0, len(images)+1)
	filtered = append(filtered, editableSceneImage{
		URL:  outputURL,
		Type: "cover",
	})
	for _, image := range images {
		if strings.TrimSpace(image.URL) == outputURL {
			continue
		}
		if strings.TrimSpace(image.Type) == "cover" {
			image.Type = "gallery"
		}
		filtered = append(filtered, image)
	}

	scene.CoverURL = outputURL
	scene.Images = encodeEditableSceneImages(filtered)
	models.AddAction(scene.SceneID, "edit", "cover_url", outputURL)
	models.AddAction(scene.SceneID, "edit", "images", scene.Images)

	if err := scene.Save(); err != nil {
		log.WithError(err).Warn("failed to save scene thumbnail changes")
		resp.WriteHeader(http.StatusBadGateway)
		return
	}

	scene.UpdateStatus()

	if err := scene.GetIfExistByPK(scene.ID); err != nil {
		log.WithError(err).Warn("failed to reload scene after thumbnail generation")
		resp.WriteHeader(http.StatusBadGateway)
		return
	}

	log.WithFields(logrus.Fields{
		"scene_id":      scene.ID,
		"video_file_id": videoFile.ID,
		"seek":          seek,
		"cover_url":     outputURL,
	}).Info("Generated scene thumbnail")

	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}
