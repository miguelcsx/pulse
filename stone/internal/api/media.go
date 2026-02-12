package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/pulse/stone/internal/middleware"
	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/service"
)

type initMediaUploadRequest struct {
	ContentType string `json:"content_type" binding:"required"`
	Filename    string `json:"filename" binding:"required"`
	MimeType    string `json:"mime_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

func (s *Server) InitMediaUpload(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req initMediaUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	asset, err := s.mediaService.InitUpload(userID, req.ContentType, req.Filename, req.MimeType, req.SizeBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"asset": asset,
		"upload": gin.H{
			"method": "PUT",
			"url":    fmt.Sprintf("/media/uploads/%s/file", asset.ID.String()),
		},
	})
}

func (s *Server) UploadMediaBinary(c *gin.Context) {
	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media asset id"})
		return
	}

	userID := middleware.GetUserID(c)
	maxBytes := int64(s.cfg.StorageMaxSizeMB) * 1024 * 1024
	contentLength := c.Request.ContentLength
	if maxBytes > 0 {
		if contentLength > maxBytes {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "payload too large"})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
	}

	asset, err := s.mediaService.UploadBinary(assetID, userID, c.Request.Body, contentLength)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMediaNotUploadable):
			c.JSON(http.StatusConflict, gin.H{"error": "media asset is not uploadable"})
		case errors.Is(err, service.ErrMediaAssetNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "media asset not found"})
		case errors.Is(err, service.ErrMediaTooLarge):
			c.JSON(http.StatusBadRequest, gin.H{"error": "file exceeds declared size"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload media"})
		}
		return
	}

	c.JSON(http.StatusAccepted, asset)
}

func (s *Server) GetMediaAsset(c *gin.Context) {
	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media asset id"})
		return
	}

	userID := middleware.GetUserID(c)
	asset, err := s.mediaService.GetForOwner(assetID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "media asset not found"})
		return
	}

	statusCode := http.StatusOK
	if asset.Status == model.MediaAssetStatusFailed {
		statusCode = http.StatusUnprocessableEntity
	}
	if asset.Status != model.MediaAssetStatusReady && asset.Status != model.MediaAssetStatusFailed {
		statusCode = http.StatusAccepted
	}

	retryAfter := ""
	if statusCode == http.StatusAccepted {
		retryAfter = strconv.Itoa(2)
		c.Header("Retry-After", retryAfter)
	}

	c.JSON(statusCode, asset)
}
