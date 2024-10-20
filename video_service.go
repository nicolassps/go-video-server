package main

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
)

type VideoError string

const (
	ErrResolutionInvalid  VideoError = "resolution_invalid"
	ErrVideoNotFound      VideoError = "video_not_found"
	ErrResolutionNotFound VideoError = "resolution_not_found"
	ErrVideoNotReady      VideoError = "video_not_ready"
)

type VideoService struct {
	Storages []FileStorage
	Database Database
}

func NewVideoService(storages []FileStorage, database Database) *VideoService {
	return &VideoService{
		Storages: storages,
		Database: database,
	}
}

func (vs *VideoService) GetVideo(ctx context.Context, videoID string) (*Video, error) {
	video, err := vs.Database.GetVideo(ctx, videoID)
	if err != nil {
		return nil, err
	}

	return &video, nil
}

func (vs *VideoService) CreateVideo(ctx context.Context, inputFilePath string) (*Video, error) {
	metadata, err := GetMetadata(inputFilePath)
	if err != nil {
		return nil, err
	}

	videoID := uuid.New().String()
	video := Video{
		ID:            videoID,
		VideoMetadata: metadata,
		Status:        VideoStatusPending,
	}

	err = vs.Database.SaveVideo(ctx, video)
	if err != nil {
		return nil, err
	}

	go func() {
		processedVideo, err := ProcessVideo(inputFilePath, videoID, vs.Storages)

		if err != nil {
			log.Printf("Error processing video: %v", err)

			video.Status = VideoStatusError
			err = vs.Database.SaveVideo(context.Background(), video)

			if err != nil {
				log.Printf("Error saving video: %v", err)
			}

			return
		}

		video.Status = VideoStatusComplete
		video.Resolutions = processedVideo.Resolutions
		err = vs.Database.SaveVideo(context.Background(), video)

		if err != nil {
			log.Printf("Error saving video: %v", err)
		}
	}()

	return &video, nil
}

func (vs *VideoService) GetVideoURL(ctx context.Context, videoID, resolution string) (string, error) {
	if !IsValidResolution(resolution) {
		return "", errors.New(string(ErrResolutionInvalid))
	}

	video, err := vs.Database.GetVideo(ctx, videoID)
	if err != nil {
		return "", err
	}

	if !video.VideoIsReady() {
		return "", errors.New(string(ErrVideoNotReady))
	}

	currentUrl := video.GetResolutionURL(resolution)
	if currentUrl != "" && !video.IsExpired(resolution) {
		return currentUrl, nil
	}

	currentResolution := video.GetResolution(resolution)
	if currentResolution == nil {
		return "", errors.New(string(ErrResolutionNotFound))
	}

	manifest, err := GenerateSegmentedManifestSigned(ctx, videoID, *currentResolution, vs.Storages[0])
	if err != nil {
		return "", err
	}

	video.AssignNewURL(resolution, manifest)
	err = vs.Database.SaveVideo(context.Background(), video)

	if err != nil {
		log.Printf("Error saving video: %v", err)
	}

	return manifest, nil
}
