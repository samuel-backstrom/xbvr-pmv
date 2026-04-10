package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/xbapps/xbvr/pkg/models"
)

const handyScriptUploadURL = "https://scripts01.handyfeeling.com/api/script/v0/temp/upload"
const handyAPIBaseURL = "https://www.handyfeeling.com/api/handy/v2"

type handyScriptAction struct {
	At  int64 `json:"at"`
	Pos int   `json:"pos"`
}

type handyScriptPayload struct {
	Actions []handyScriptAction `json:"actions"`
}

func readHandyFileData(file models.File) ([]byte, error) {
	switch file.Volume.Type {
	case "local":
		return os.ReadFile(file.GetPath())
	case "putio":
		id, err := strconv.ParseInt(file.Path, 10, 64)
		if err != nil {
			return nil, err
		}

		client := file.Volume.GetPutIOClient()
		url, err := client.Files.URL(context.Background(), id, false)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("putio download failed: %s", resp.Status)
		}

		return io.ReadAll(resp.Body)
	default:
		return nil, fmt.Errorf("unsupported volume type: %s", file.Volume.Type)
	}
}

func convertHandyScriptDataToCSV(scriptData []byte) ([]byte, error) {
	var payload handyScriptPayload
	if err := json.Unmarshal(scriptData, &payload); err == nil && len(payload.Actions) > 0 {
		sort.SliceStable(payload.Actions, func(i, j int) bool {
			return payload.Actions[i].At < payload.Actions[j].At
		})

		var csv bytes.Buffer
		csv.WriteString("#Created by Handy SDK v2\n")
		for _, action := range payload.Actions {
			if action.At < 0 {
				action.At = 0
			}
			fmt.Fprintf(&csv, "%d,%d\n", action.At, action.Pos)
		}
		return csv.Bytes(), nil
	}

	if isValidHandyCSV(scriptData) {
		return scriptData, nil
	}

	return nil, fmt.Errorf("input is not valid funscript JSON or Handy CSV")
}

func isValidHandyCSV(scriptData []byte) bool {
	lines := strings.Split(strings.TrimSpace(string(scriptData)), "\n")
	dataLines := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			return false
		}

		if _, err := strconv.Atoi(strings.TrimSpace(parts[0])); err != nil {
			return false
		}
		if _, err := strconv.Atoi(strings.TrimSpace(parts[1])); err != nil {
			return false
		}

		dataLines++
	}

	return dataLines >= 2
}

func uploadHandyScriptData(scriptData []byte, filename string) (string, error) {
	csvData, err := convertHandyScriptDataToCSV(scriptData)
	if err != nil {
		return "", err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(csvData); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, handyScriptUploadURL, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("handy script upload failed: %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var out struct {
		URL   string `json:"url"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.URL == "" {
		if out.Error != "" {
			return "", fmt.Errorf("handy script upload failed: %s", out.Error)
		}
		return "", fmt.Errorf("handy script upload returned empty url")
	}

	return out.URL, nil
}

func handyRequestWithKey(connectionKey string) *resty.Request {
	return resty.New().
		SetTimeout(10*time.Second).
		R().
		SetHeader("X-Connection-Key", connectionKey)
}

func handyConnected(connectionKey string) (bool, error) {
	resp, err := handyRequestWithKey(connectionKey).Get(handyAPIBaseURL + "/connected")
	if err != nil {
		return false, err
	}
	if !resp.IsSuccess() {
		return false, fmt.Errorf("handy connected check failed: %s", resp.Status())
	}

	var out struct {
		Connected bool `json:"connected"`
	}
	if err := json.Unmarshal([]byte(resp.String()), &out); err != nil {
		return false, err
	}
	return out.Connected, nil
}

func handySetOffset(connectionKey string, offsetMs int) error {
	if offsetMs == 0 {
		return nil
	}

	resp, err := handyRequestWithKey(connectionKey).
		SetBody(map[string]int{"offset": offsetMs}).
		Put(handyAPIBaseURL + "/hstp/offset")
	if err != nil {
		return err
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("handy offset update failed: %s", resp.Status())
	}
	return nil
}

func handySetupScript(connectionKey, scriptURL string) error {
	resp, err := handyRequestWithKey(connectionKey).
		SetBody(map[string]string{"url": scriptURL}).
		Put(handyAPIBaseURL + "/hssp/setup")
	if err != nil {
		return err
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("handy setup failed: %s", resp.Status())
	}
	return nil
}

func handyPlayScript(connectionKey string, startTimeMs int) error {
	resp, err := handyRequestWithKey(connectionKey).
		SetBody(map[string]int{
			"startTime":           startTimeMs,
			"estimatedServerTime": int(time.Now().UnixMilli()),
		}).
		Put(handyAPIBaseURL + "/hssp/play")
	if err != nil {
		return err
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("handy play failed: %s", resp.Status())
	}
	return nil
}

func handyStopScript(connectionKey string) error {
	resp, err := handyRequestWithKey(connectionKey).
		Put(handyAPIBaseURL + "/hssp/stop")
	if err != nil {
		return err
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("handy stop failed: %s", resp.Status())
	}
	return nil
}
