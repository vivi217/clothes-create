package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"garment-ai/backend/internal/config"
)

func RegisterRoutes(router *gin.Engine, cfg config.Config) {
	router.GET("/healthz", func(ctx *gin.Context) {
		writeSuccess(ctx, gin.H{
			"service":     "backend",
			"status":      "ok",
			"environment": cfg.AppEnv,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	router.GET("/api/v1/system/health", func(ctx *gin.Context) {
		writeSuccess(ctx, gin.H{
			"postgresDsn":   cfg.PostgresDSN,
			"redisAddr":     cfg.RedisAddr,
			"minioEndpoint": cfg.MinioEndpoint,
			"engineBaseUrl": cfg.EngineBaseURL,
			"milvusAddress": cfg.MilvusAddress,
		})
	})

	router.POST("/api/v1/upload/photos", func(ctx *gin.Context) {
		var request UploadPhotosRequest
		if err := ctx.ShouldBindJSON(&request); err != nil {
			writeError(ctx, http.StatusBadRequest, CodeInvalidRequest, err.Error())
			return
		}

		storedPhotos, err := uploadPhotosToMinio(ctx.Request.Context(), cfg, request)
		if err != nil {
			writeError(ctx, http.StatusInternalServerError, CodeUploadFailed, err.Error())
			return
		}

		uploaded := make([]gin.H, 0, len(storedPhotos))
		for _, photo := range storedPhotos {
			uploaded = append(uploaded, gin.H{
				"view":      photo.View,
				"fileName":  photo.FileName,
				"objectKey": photo.ObjectKey,
				"url":       photo.URL,
			})
		}

		writeSuccess(ctx, gin.H{
			"sessionId": request.SessionID,
			"bucket":    cfg.MinioBucket,
			"photos":    uploaded,
		})
	})

	router.POST("/api/v1/params/generate", func(ctx *gin.Context) {
		var request GenerateParamsRequest
		if err := ctx.ShouldBindJSON(&request); err != nil {
			writeError(ctx, http.StatusBadRequest, CodeInvalidRequest, err.Error())
			return
		}

		writeSuccess(ctx, gin.H{
			"templateId": request.TemplateID,
			"structure":  request.Structure,
			"ratios":     request.Ratios,
			"params":     sampleGarmentParams(),
		})
	})

	router.POST("/api/v1/3d/generate", func(ctx *gin.Context) {
		var request GeneratePreviewRequest
		if err := ctx.ShouldBindJSON(&request); err != nil {
			writeError(ctx, http.StatusBadRequest, CodeInvalidRequest, err.Error())
			return
		}

		engineResponse, err := requestEnginePreview(cfg, request.Params)
		if err != nil {
			writeError(ctx, http.StatusBadGateway, CodePreviewGenerateError, err.Error())
			return
		}

		writeSuccess(ctx, gin.H{
			"taskId":      fmt.Sprintf("preview-%d", time.Now().UTC().Unix()),
			"params":      engineResponse.Data.Params,
			"glbUrl":      previewProxyURL(ctx, engineResponse.Data.AssetKey),
			"cacheHit":    engineResponse.Data.CacheHit,
			"vertexCount": engineResponse.Data.VertexCount,
			"fileSizeKb":  engineResponse.Data.FileSizeKB,
		})
	})

	router.GET("/api/v1/3d/assets/*object_path", func(ctx *gin.Context) {
		objectKey := strings.TrimPrefix(ctx.Param("object_path"), "/")
		if !strings.HasPrefix(objectKey, "previews/generated/") || !strings.HasSuffix(strings.ToLower(objectKey), ".glb") {
			writeError(ctx, http.StatusBadRequest, CodeInvalidRequest, "invalid preview asset path")
			return
		}

		object, info, err := getObjectFromMinio(ctx.Request.Context(), cfg, objectKey)
		if err != nil {
			writeError(ctx, http.StatusNotFound, CodePreviewGenerateError, err.Error())
			return
		}
		defer object.Close()

		contentType := info.ContentType
		if contentType == "" {
			contentType = "model/gltf-binary"
		}
		ctx.Header("Content-Type", contentType)
		ctx.Header("Content-Length", fmt.Sprintf("%d", info.Size))
		ctx.Header("Cache-Control", "public, max-age=1800")
		ctx.Status(http.StatusOK)
		_, _ = io.Copy(ctx.Writer, object)
	})

	router.POST("/api/v1/pattern/generate", func(ctx *gin.Context) {
		var request GeneratePatternRequest
		if err := ctx.ShouldBindJSON(&request); err != nil {
			writeError(ctx, http.StatusBadRequest, CodeInvalidRequest, err.Error())
			return
		}

		patternResult, err := generateIndustrialPattern(ctx.Request.Context(), cfg, request.Params)
		if err != nil {
			writeError(ctx, http.StatusBadGateway, CodePatternGenerateError, err.Error())
			return
		}

		writeSuccess(ctx, gin.H{
			"taskId":      fmt.Sprintf("pattern-%d", time.Now().UTC().Unix()),
			"params":      request.Params,
			"patternJson": patternResult.PatternJSON,
			"fileId":      patternResult.FileID,
			"dxfUrl":      dxfAssetURL(ctx, patternResult.FileID),
		})
	})

	router.GET("/api/v1/files/assets/dxf/:file_id", func(ctx *gin.Context) {
		fileID := strings.TrimSpace(ctx.Param("file_id"))
		if fileID == "" {
			writeError(ctx, http.StatusBadRequest, CodeInvalidRequest, "invalid dxf file id")
			return
		}

		objectKey := fmt.Sprintf("patterns/generated/%s.dxf", fileID)
		object, info, err := getObjectFromMinio(ctx.Request.Context(), cfg, objectKey)
		if err != nil {
			writeError(ctx, http.StatusNotFound, CodeFileNotFound, err.Error())
			return
		}
		defer object.Close()

		contentType := info.ContentType
		if contentType == "" {
			contentType = "application/dxf"
		}
		ctx.Header("Content-Type", contentType)
		ctx.Header("Content-Length", fmt.Sprintf("%d", info.Size))
		ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.dxf", fileID))
		ctx.Header("Cache-Control", "public, max-age=1800")
		ctx.Status(http.StatusOK)
		_, _ = io.Copy(ctx.Writer, object)
	})

	router.GET("/api/v1/files/download/dxf/:file_id", func(ctx *gin.Context) {
		fileID := ctx.Param("file_id")
		objectKey := fmt.Sprintf("patterns/generated/%s.dxf", fileID)
		if _, _, err := getObjectFromMinio(ctx.Request.Context(), cfg, objectKey); err != nil {
			writeError(ctx, http.StatusNotFound, CodeFileNotFound, err.Error())
			return
		}

		writeSuccess(ctx, gin.H{
			"fileId":      fileID,
			"fileName":    fmt.Sprintf("%s.dxf", fileID),
			"format":      "ASTM DXF 2000",
			"downloadUrl": dxfAssetURL(ctx, fileID),
		})
	})
}

func previewProxyURL(ctx *gin.Context, objectKey string) string {
	scheme := "http"
	if ctx.Request.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := ctx.GetHeader("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	}
	return fmt.Sprintf("%s://%s/api/v1/3d/assets/%s", scheme, ctx.Request.Host, objectKey)
}

func dxfAssetURL(ctx *gin.Context, fileID string) string {
	scheme := "http"
	if ctx.Request.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := ctx.GetHeader("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	}
	return fmt.Sprintf("%s://%s/api/v1/files/assets/dxf/%s", scheme, ctx.Request.Host, fileID)
}