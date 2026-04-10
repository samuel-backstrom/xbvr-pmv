package tasks

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/djherbis/times"
	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/ffprobe"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
)

const (
	maxPMVImportBatchLimit           = 100
	defaultPMVImportBatchConcurrency = 3
	maxPMVImportBatchConcurrency     = 10
)

type PMVImportRequest struct {
	URL         string `json:"url,omitempty"`
	ListURL     string `json:"list_url,omitempty"`
	PathPrefix  string `json:"path_prefix,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Concurrency int    `json:"concurrency,omitempty"`
}

type PMVImportResult struct {
	URL                string `json:"url"`
	SceneURL           string `json:"scene_url,omitempty"`
	MediaURL           string `json:"media_url,omitempty"`
	DownloadedPath     string `json:"downloaded_path,omitempty"`
	FileID             uint   `json:"file_id,omitempty"`
	SceneID            string `json:"scene_id,omitempty"`
	FunscriptGenerated bool   `json:"funscript_generated,omitempty"`
	Skipped            bool   `json:"skipped,omitempty"`
	Message            string `json:"message,omitempty"`
}

type PMVImportBatchItem struct {
	Rank   int              `json:"rank"`
	URL    string           `json:"url"`
	Result *PMVImportResult `json:"result,omitempty"`
	Error  string           `json:"error,omitempty"`
}

type PMVImportBatchResult struct {
	ListURL         string               `json:"list_url"`
	Requested       int                  `json:"requested"`
	Queued          int                  `json:"queued"`
	Imported        int                  `json:"imported"`
	SkippedExisting int                  `json:"skipped_existing"`
	Funscripts      int                  `json:"funscripts_generated"`
	Errors          int                  `json:"errors"`
	Results         []PMVImportBatchItem `json:"results"`
}

type pmvImportRuntime struct {
	Volume       models.Volume
	Destination  string
	SkipExisting bool
}

func ImportPMVHavenVideo(req PMVImportRequest) (*PMVImportResult, int, error) {
	sceneURL := strings.TrimSpace(req.URL)
	if sceneURL == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("url is required")
	}

	db, _ := models.GetDB()
	defer db.Close()

	runtime, err := resolvePMVImportRuntime(db, strings.TrimSpace(req.PathPrefix), false)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	return importSinglePMVHavenVideo(sceneURL, runtime)
}

func ImportPMVHavenList(req PMVImportRequest) (*PMVImportBatchResult, int, error) {
	listURL := strings.TrimSpace(req.ListURL)
	if listURL == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("list_url is required")
	}

	limit := normalizePMVImportBatchLimit(req.Limit)
	concurrency := normalizePMVImportBatchConcurrency(req.Concurrency)

	candidates, err := scrape.FetchPMVHavenListingCandidates(listURL, limit)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}

	db, _ := models.GetDB()
	defer db.Close()

	runtime, err := resolvePMVImportRuntime(db, strings.TrimSpace(req.PathPrefix), true)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	type job struct {
		Index int
		URL   string
	}
	type outcome struct {
		Index  int
		Result *PMVImportResult
		Err    error
	}

	results := make([]PMVImportBatchItem, len(candidates))
	jobs := make(chan job)
	outcomes := make(chan outcome, len(candidates))
	var wg sync.WaitGroup

	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				result, _, err := importSinglePMVHavenVideo(job.URL, runtime)
				outcomes <- outcome{
					Index:  job.Index,
					Result: result,
					Err:    err,
				}
			}
		}()
	}

	for i, candidate := range candidates {
		results[i] = PMVImportBatchItem{
			Rank: i + 1,
			URL:  strings.TrimSpace(candidate.SceneURL),
		}
		jobs <- job{Index: i, URL: strings.TrimSpace(candidate.SceneURL)}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(outcomes)
	}()

	batch := &PMVImportBatchResult{
		ListURL:   listURL,
		Requested: limit,
		Queued:    len(candidates),
		Results:   results,
	}
	if limit <= 0 {
		batch.Requested = len(candidates)
	}

	for outcome := range outcomes {
		item := &batch.Results[outcome.Index]
		if outcome.Err != nil {
			item.Error = outcome.Err.Error()
			batch.Errors++
			continue
		}
		item.Result = outcome.Result
		if outcome.Result == nil {
			batch.Errors++
			item.Error = "empty import result"
			continue
		}
		if outcome.Result.Skipped {
			batch.SkippedExisting++
			continue
		}
		batch.Imported++
		if outcome.Result.FunscriptGenerated {
			batch.Funscripts++
		}
	}

	return batch, http.StatusOK, nil
}

func importSinglePMVHavenVideo(sceneURL string, runtime pmvImportRuntime) (*PMVImportResult, int, error) {
	sceneURL = strings.TrimSpace(sceneURL)
	if sceneURL == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("url is required")
	}

	tlog := log.WithField("task", "pmv-import").WithField("url", sceneURL)
	videoMeta, err := scrape.FetchPMVHavenVideoMetadata(sceneURL)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}

	if err := os.MkdirAll(runtime.Destination, 0o755); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	destPath := filepath.Join(runtime.Destination, filepath.Base(videoMeta.Filename))
	if runtime.SkipExisting {
		if existingPath := findExistingPMVImportPath(videoMeta.SceneURL, runtime.Volume.ID); existingPath != "" {
			tlog.Infof("skipped existing scene_url=%q video=%q", videoMeta.SceneURL, existingPath)
			return &PMVImportResult{
				URL:            sceneURL,
				SceneURL:       videoMeta.SceneURL,
				MediaURL:       videoMeta.MediaURL,
				DownloadedPath: existingPath,
				Skipped:        true,
				Message:        "skipped: PMVHaven scene already imported",
			}, http.StatusOK, nil
		}
	}
	if runtime.SkipExisting && fileExistsNonEmpty(destPath) {
		tlog.Infof("skipped existing video=%q", destPath)
		return &PMVImportResult{
			URL:            sceneURL,
			SceneURL:       videoMeta.SceneURL,
			MediaURL:       videoMeta.MediaURL,
			DownloadedPath: destPath,
			Skipped:        true,
			Message:        "skipped: video already exists",
		}, http.StatusOK, nil
	}

	if !fileExistsNonEmpty(destPath) {
		tlog.Infof("downloading media_url=%q dest=%q", videoMeta.MediaURL, destPath)
		if err := downloadPMVVideoFile(videoMeta.MediaURL, destPath); err != nil {
			return nil, http.StatusBadGateway, err
		}
	} else {
		tlog.Infof("download skipped existing_file=%q", destPath)
	}

	videoFile, err := scanLocalVideoFile(destPath, runtime.Volume.ID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	db, _ := models.GetDB()
	defer db.Close()

	candidate := PMVMatchCandidate{
		Title:        strings.TrimSpace(videoMeta.Title),
		SceneURL:     strings.TrimSpace(videoMeta.SceneURL),
		ThumbnailURL: strings.TrimSpace(videoMeta.ThumbnailURL),
		Channel:      strings.TrimSpace(videoMeta.Channel),
		Description:  strings.TrimSpace(videoMeta.Description),
	}
	sceneID, err := applyPMVMatch(db, &videoFile, candidate)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	pyResult, statusCode, err := GeneratePythonDancerFunscripts(PythonDancerBatchRequest{
		FileID:          videoFile.ID,
		Concurrency:     1,
		ForceRegenerate: false,
		PostProcessMode: postProcessModeAuto,
	})
	if err != nil {
		return nil, statusCode, err
	}

	funscriptGenerated := pyResult != nil && pyResult.Generated > 0 && pyResult.Errors == 0
	message := "PMV imported, scene created, and funscript generated"
	if !funscriptGenerated {
		message = "PMV imported and scene created"
	}

	return &PMVImportResult{
		URL:                sceneURL,
		SceneURL:           videoMeta.SceneURL,
		MediaURL:           videoMeta.MediaURL,
		DownloadedPath:     destPath,
		FileID:             videoFile.ID,
		SceneID:            sceneID,
		FunscriptGenerated: funscriptGenerated,
		Message:            message,
	}, http.StatusOK, nil
}

func findExistingPMVImportPath(sceneURL string, volumeID uint) string {
	sceneURL = strings.TrimSpace(sceneURL)
	if sceneURL == "" {
		return ""
	}

	db, _ := models.GetDB()
	defer db.Close()

	var scene models.Scene
	err := db.Where("scene_url = ? OR member_url = ?", sceneURL, sceneURL).First(&scene).Error
	if err != nil {
		return ""
	}

	files, err := scene.GetFiles()
	if err != nil {
		return ""
	}
	for _, file := range files {
		if file.Type != "video" {
			continue
		}
		if volumeID != 0 && file.VolumeID != volumeID {
			continue
		}
		if file.Exists() {
			return file.GetPath()
		}
	}
	return ""
}

func resolvePMVImportRuntime(db *gorm.DB, pathPrefix string, skipExisting bool) (pmvImportRuntime, error) {
	volume, destination, err := resolvePMVImportDestination(db, pathPrefix)
	if err != nil {
		return pmvImportRuntime{}, err
	}
	return pmvImportRuntime{
		Volume:       volume,
		Destination:  destination,
		SkipExisting: skipExisting,
	}, nil
}

func normalizePMVImportBatchLimit(limit int) int {
	if limit <= 0 {
		return 0
	}
	if limit > maxPMVImportBatchLimit {
		return maxPMVImportBatchLimit
	}
	return limit
}

func normalizePMVImportBatchConcurrency(concurrency int) int {
	if concurrency <= 0 {
		return defaultPMVImportBatchConcurrency
	}
	if concurrency > maxPMVImportBatchConcurrency {
		return maxPMVImportBatchConcurrency
	}
	return concurrency
}

func resolvePMVImportDestination(db *gorm.DB, pathPrefix string) (models.Volume, string, error) {
	var volumes []models.Volume
	if err := db.Where("type = ?", "local").Order("path asc").Find(&volumes).Error; err != nil {
		return models.Volume{}, "", err
	}
	if len(volumes) == 0 {
		return models.Volume{}, "", fmt.Errorf("no local volume configured")
	}

	if pathPrefix == "" {
		return volumes[0], volumes[0].Path, nil
	}

	cleanPrefix := filepath.Clean(pathPrefix)
	for _, vol := range volumes {
		volPath := filepath.Clean(vol.Path)
		if cleanPrefix == volPath || strings.HasPrefix(cleanPrefix+string(os.PathSeparator), volPath+string(os.PathSeparator)) {
			return vol, cleanPrefix, nil
		}
	}
	return models.Volume{}, "", fmt.Errorf("path_prefix %q does not belong to a configured local volume", pathPrefix)
}

func downloadPMVVideoFile(sourceURL, destPath string) error {
	resp, err := http.Get(sourceURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tmpPath := destPath + ".part"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, destPath)
}

func scanLocalVideoFile(path string, volID uint) (models.File, error) {
	var fl models.File
	db, _ := models.GetDB()
	defer db.Close()

	fStat, err := os.Stat(path)
	if err != nil {
		return fl, err
	}
	fTimes, err := times.Stat(path)
	if err != nil {
		return fl, err
	}

	var createdAt time.Time
	if fTimes.HasBirthTime() {
		createdAt = fTimes.BirthTime()
	} else {
		createdAt = fTimes.ModTime()
	}

	db.Where(&models.File{
		Path:     filepath.Dir(path),
		Filename: filepath.Base(path),
		Type:     "video",
	}).FirstOrCreate(&fl)

	fl.Size = fStat.Size()
	fl.CreatedTime = createdAt
	fl.UpdatedTime = fTimes.ModTime()
	fl.VolumeID = volID

	if hash, err := Hash(path); err == nil {
		fl.OsHash = fmt.Sprintf("%x", hash)
	}

	if ffdata, err := ffprobe.GetProbeData(path, 10*time.Second); err == nil {
		if vs := ffdata.GetFirstVideoStream(); vs != nil {
			if vs.BitRate != "" {
				if bitRate, convErr := strconv.Atoi(vs.BitRate); convErr == nil {
					fl.VideoBitRate = bitRate
				}
			}
			fl.VideoAvgFrameRate = vs.AvgFrameRate
			fl.VideoCodecName = vs.CodecName
			fl.VideoWidth = vs.Width
			fl.VideoHeight = vs.Height
			if dur, durErr := strconv.ParseFloat(vs.Duration, 64); durErr == nil {
				fl.VideoDuration = dur
			} else if ffdata.Format.DurationSeconds > 0 {
				fl.VideoDuration = ffdata.Format.DurationSeconds
			}
			fl.CalculateFramerate()
		}
	}

	if err := fl.Save(); err != nil {
		return fl, err
	}
	return fl, nil
}
