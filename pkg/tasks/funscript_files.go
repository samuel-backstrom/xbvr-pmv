package tasks

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/djherbis/times"
	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/models"
)

func siblingFunscriptPath(videoPath string) string {
	return strings.TrimSuffix(videoPath, filepath.Ext(videoPath)) + ".funscript"
}

func fileExistsNonEmpty(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

func escapedFilenameLike(s string) string {
	var buffer bytes.Buffer
	json.HTMLEscape(&buffer, []byte(s))
	return buffer.String()
}

func appendSceneFilename(scene *models.Scene, filename string) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return
	}

	var filenames []string
	_ = json.Unmarshal([]byte(scene.FilenamesArr), &filenames)
	for _, existing := range filenames {
		if existing == filename {
			return
		}
	}
	filenames = append(filenames, filename)
	if data, err := json.Marshal(filenames); err == nil {
		scene.FilenamesArr = string(data)
	}
}

func linkScriptToScene(db *gorm.DB, video *models.File, script *models.File) (string, error) {
	if video.SceneID != 0 {
		script.SceneID = video.SceneID
		if err := script.Save(); err != nil {
			return "", err
		}
		var scene models.Scene
		if err := scene.GetIfExistByPK(video.SceneID); err == nil {
			appendSceneFilename(&scene, video.Filename)
			appendSceneFilename(&scene, script.Filename)
			if err := scene.Save(); err != nil {
				return "", err
			}
			scene.UpdateStatus()
			return scene.SceneID, nil
		}
		return "", nil
	}

	filename := escapedFilenameLike(video.Filename)
	var scenes []models.Scene
	if err := db.Where("filenames_arr LIKE ?", `%"`+filename+`"%`).Find(&scenes).Error; err != nil {
		return "", err
	}
	if len(scenes) != 1 {
		return "", nil
	}

	video.SceneID = scenes[0].ID
	if err := video.Save(); err != nil {
		return "", err
	}
	script.SceneID = scenes[0].ID
	if err := script.Save(); err != nil {
		return "", err
	}

	appendSceneFilename(&scenes[0], video.Filename)
	appendSceneFilename(&scenes[0], script.Filename)
	if err := scenes[0].Save(); err != nil {
		return "", err
	}
	scenes[0].UpdateStatus()
	return scenes[0].SceneID, nil
}

func scanScriptFile(path string, volID uint) (models.File, error) {
	db, _ := models.GetDB()
	defer db.Close()

	var fl models.File
	db.Where(&models.File{
		Path:     filepath.Dir(path),
		Filename: filepath.Base(path),
		Type:     "script",
	}).FirstOrCreate(&fl)

	fStat, err := os.Stat(path)
	if err != nil {
		return fl, err
	}
	fTimes, err := times.Stat(path)
	if err != nil {
		return fl, err
	}

	fl.Size = fStat.Size()
	fl.CreatedTime = fTimes.ModTime()
	fl.UpdatedTime = fTimes.ModTime()
	fl.VolumeID = volID
	fl.HasHeatmap = false
	if duration, err := getFunscriptDuration(path); err == nil {
		fl.VideoDuration = duration
	}
	if err := fl.Save(); err != nil {
		return fl, err
	}
	return fl, nil
}
