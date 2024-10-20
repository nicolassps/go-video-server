package main

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type API struct {
	VideoService VideoService
}

func NewAPI(videoService VideoService) *API {
	return &API{
		VideoService: videoService,
	}
}

func (api *API) HandleUpload(c *gin.Context) {
	file, err := c.FormFile("video")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Erro ao receber arquivo: %v", err))
		return
	}

	inputFilePath := filepath.Join("uploads", file.Filename)
	if err := c.SaveUploadedFile(file, inputFilePath); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Erro ao salvar o arquivo: %v", err))
		return
	}

	video, err := api.VideoService.CreateVideo(c, inputFilePath)

	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Erro ao criar vídeo: %v", err))
		return
	}

	c.JSON(http.StatusOK, video)
}

func (api *API) GetVideo(c *gin.Context) {
	fmt.Println("GetVideo")
	videoID := c.Param("id")

	fmt.Println("videoID", videoID)
	video, err := api.VideoService.GetVideo(c, videoID)

	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Erro ao buscar vídeo: %v", err))
		return
	}

	c.JSON(http.StatusOK, video)
}

func (api *API) GetVideoURL(c *gin.Context) {
	videoID := c.Param("id")
	resolution := c.Query("resolution")

	url, err := api.VideoService.GetVideoURL(c, videoID, resolution)

	if err != nil {
		message := err.Error()

		if message == string(ErrVideoNotFound) {
			c.String(http.StatusNotFound, fmt.Sprintf("Video not found: %v", err))
			return
		}

		if message == string(ErrResolutionNotFound) {
			c.String(http.StatusBadRequest, fmt.Sprintf("Resolution not found: %v", err))
			return
		}

		if message == string(ErrVideoNotReady) {
			c.String(http.StatusConflict, fmt.Sprintf("Video not ready: %v", err))
			return
		}

		if message == string(ErrResolutionInvalid) {
			c.String(http.StatusBadRequest, fmt.Sprintf("Resolution invalid: %v", err))
			return
		}

		c.String(http.StatusInternalServerError, fmt.Sprintf("Error searching video: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}
