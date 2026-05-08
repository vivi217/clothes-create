package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"garment-ai/backend/internal/config"
)

type enginePreviewResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Params      map[string]any `json:"params"`
			GlbURL      string         `json:"glbUrl"` // Updated to match the new structure
		VertexCount int            `json:"vertexCount"`
		FileSizeKB  int            `json:"fileSizeKb"`
		CacheHit    bool           `json:"cacheHit"`
		AssetKey    string         `json:"assetKey"`
	} `json:"data"`
}

func requestEnginePreview(cfg config.Config, params map[string]any) (enginePreviewResponse, error) {
	var result enginePreviewResponse
	payload, err := json.Marshal(map[string]any{"params": params})
	if err != nil {
		return result, fmt.Errorf("marshal engine payload: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, strings.TrimRight(cfg.EngineBaseURL, "/")+"/api/v1/engine/garmentcode/3d", bytes.NewReader(payload))
	if err != nil {
		return result, fmt.Errorf("create engine request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 45 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return result, fmt.Errorf("call engine service: %w", err)
	}
	defer response.Body.Close()

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("decode engine response: %w", err)
	}

	if response.StatusCode != http.StatusOK || result.Code != CodeSuccess {
		if result.Message == "" {
			result.Message = "engine preview generation failed"
		}
		return result, fmt.Errorf(result.Message)
	}

	return result, nil
}

func publicAssetURL(cfg config.Config, objectKey string) string {
	return fmt.Sprintf("http://%s/%s/%s", cfg.MinioPublicEndpoint, cfg.MinioBucket, objectKey)
}