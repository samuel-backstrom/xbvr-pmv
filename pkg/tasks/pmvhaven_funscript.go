package tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
)

func importPMVHavenFunscript(videoMeta scrape.PMVHavenVideoMetadata, videoFile *models.File) (bool, error) {
	if videoFile == nil || videoFile.ID == 0 {
		return false, nil
	}

	scriptURL := strings.TrimSpace(videoMeta.FunscriptURL)
	if scriptURL == "" {
		return false, nil
	}

	targetPath := siblingFunscriptPath(videoFile.GetPath())
	if fileExistsNonEmpty(targetPath) {
		scriptFile, err := scanScriptFile(targetPath, videoFile.VolumeID)
		if err != nil {
			return false, err
		}
		db, _ := models.GetDB()
		defer db.Close()

		if _, err := linkScriptToScene(db, videoFile, &scriptFile); err != nil {
			return false, err
		}
		return true, nil
	}

	rawData, err := downloadPMVHavenScriptData(scriptURL)
	if err != nil {
		return false, err
	}

	scriptData, err := convertPMVHavenScriptDataToFunscript(rawData, videoFile.VideoDuration)
	if err != nil {
		return false, err
	}

	tmpPath := targetPath + ".part"
	if err := os.WriteFile(tmpPath, scriptData, 0o644); err != nil {
		return false, err
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return false, err
	}

	scriptFile, err := scanScriptFile(targetPath, videoFile.VolumeID)
	if err != nil {
		_ = os.Remove(targetPath)
		return false, err
	}

	db, _ := models.GetDB()
	defer db.Close()

	if _, err := linkScriptToScene(db, videoFile, &scriptFile); err != nil {
		_ = os.Remove(targetPath)
		return false, err
	}

	return true, nil
}

func downloadPMVHavenScriptData(sourceURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", scrape.UserAgent)

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("pmvhaven funscript fetch failed with status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func convertPMVHavenScriptDataToFunscript(scriptData []byte, videoDurationSeconds float64) ([]byte, error) {
	var funscript Script
	if err := json.Unmarshal(scriptData, &funscript); err == nil && len(funscript.Actions) > 0 {
		return normalizePMVHavenScript(funscript, videoDurationSeconds)
	}

	actions, err := parsePMVHavenHandyCSV(scriptData)
	if err != nil {
		return nil, err
	}

	funscript = Script{
		Version: "1.0",
		Range:   100,
		Actions: actions,
	}
	if videoDurationSeconds > 0 {
		funscript.Metadata = &ScriptMetadata{Duration: int64(math.Round(videoDurationSeconds))}
	}
	return json.Marshal(funscript)
}

func normalizePMVHavenScript(funscript Script, videoDurationSeconds float64) ([]byte, error) {
	if funscript.Version == nil || strings.TrimSpace(fmt.Sprint(funscript.Version)) == "" {
		funscript.Version = "1.0"
	}
	if funscript.Range == 0 {
		funscript.Range = 100
	}
	if len(funscript.Actions) == 0 {
		return nil, fmt.Errorf("funscript actions list is empty")
	}

	sort.SliceStable(funscript.Actions, func(i, j int) bool {
		return funscript.Actions[i].At < funscript.Actions[j].At
	})
	for i := range funscript.Actions {
		if funscript.Actions[i].At < 0 {
			funscript.Actions[i].At = 0
		}
		if funscript.Actions[i].Pos < 0 {
			funscript.Actions[i].Pos = 0
		}
		if funscript.Actions[i].Pos > 100 {
			funscript.Actions[i].Pos = 100
		}
	}

	if videoDurationSeconds > 0 {
		dur := int64(math.Round(videoDurationSeconds))
		if funscript.Metadata == nil {
			funscript.Metadata = &ScriptMetadata{Duration: dur}
		} else if funscript.Metadata.Duration <= 0 {
			funscript.Metadata.Duration = dur
		}
	}

	return json.Marshal(funscript)
}

func parsePMVHavenHandyCSV(scriptData []byte) ([]Action, error) {
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(string(scriptData)), "\r\n", "\n"), "\n")
	actions := make([]Action, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("input is not valid Handy CSV")
		}

		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])

		at, err := strconv.ParseInt(left, 10, 64)
		if err != nil {
			if len(actions) == 0 && isPMVHavenCSVHeaderLine(left, right) {
				continue
			}
			return nil, fmt.Errorf("input is not valid Handy CSV")
		}

		pos, err := strconv.Atoi(right)
		if err != nil {
			return nil, fmt.Errorf("input is not valid Handy CSV")
		}

		if at < 0 {
			at = 0
		}
		if pos < 0 {
			pos = 0
		}
		if pos > 100 {
			pos = 100
		}
		actions = append(actions, Action{At: at, Pos: pos})
	}

	if len(actions) == 0 {
		return nil, fmt.Errorf("input is not valid Handy CSV")
	}

	sort.SliceStable(actions, func(i, j int) bool {
		return actions[i].At < actions[j].At
	})
	return actions, nil
}

func isPMVHavenCSVHeaderLine(left, right string) bool {
	switch strings.ToLower(strings.TrimSpace(left)) {
	case "at", "time", "timestamp":
		switch strings.ToLower(strings.TrimSpace(right)) {
		case "pos", "position", "value":
			return true
		}
	}
	return false
}
