package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/martian/v3/log"
)

type VideoStatus string

const (
	VideoStatusPending  VideoStatus = "pending"
	VideoStatusComplete VideoStatus = "complete"
	VideoStatusError    VideoStatus = "error"
)

type Video struct {
	ID            string
	VideoMetadata VideoMetadata
	Status        VideoStatus
	Resolutions   []Resolution
}

type Resolution struct {
	Resolution        string
	Manifest          string
	TotalSegments     int
	Url               string
	UrlExpirationTime time.Time
}

type FileStorage interface {
	Store(filePath string, fileContent []byte) error
	SignedURL(filePath string) (string, error)
}

type VideoMetadata struct {
	Width    int
	Height   int
	Name     string
	Duration string
}

type VideoUploadResponse struct {
	Resolutions []Resolution
}

func (v *Video) GetResolutionURL(resolution string) string {
	for _, r := range v.Resolutions {
		if r.Resolution == resolution {
			return r.Url
		}
	}

	return ""
}

func (v *Video) GetResolution(resolution string) *Resolution {
	for _, r := range v.Resolutions {
		if r.Resolution == resolution {
			return &r
		}
	}

	return nil
}

func (v *Video) VideoIsReady() bool {
	return v.Status == VideoStatusComplete
}

func (v *Video) AssignNewURL(resolution string, url string) {
	for i, r := range v.Resolutions {
		if r.Resolution == resolution {
			v.Resolutions[i].Url = url
			v.Resolutions[i].UrlExpirationTime = time.Now().Add(time.Minute * 60)
			return
		}
	}
}

func (v *Video) IsExpired(resolution string) bool {
	for _, r := range v.Resolutions {
		if r.Resolution == resolution {
			return time.Now().After(r.UrlExpirationTime)
		}
	}
	return true
}

func GetMetadata(inputFilePath string) (VideoMetadata, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "stream=width,height,duration", "-of", "default=noprint_wrappers=1:nokey=1", inputFilePath)

	var outBuffer bytes.Buffer
	cmd.Stdout = &outBuffer

	err := cmd.Run()
	if err != nil {
		return VideoMetadata{}, fmt.Errorf("metadata loading error: %v", err)
	}

	lines := strings.Split(outBuffer.String(), "\n")
	if len(lines) < 3 {
		return VideoMetadata{}, fmt.Errorf("metadata loading error: unexpected output")
	}

	width := 0
	height := 0
	duration := ""
	for i, line := range lines {
		line = strings.TrimSpace(line) // Remove espaços em branco e caracteres de controle

		if i == 0 {
			width, err = strconv.Atoi(line)
			if err != nil {
				return VideoMetadata{}, fmt.Errorf("width parse error: %v", err)
			}
		} else if i == 1 {
			height, err = strconv.Atoi(line)
			if err != nil {
				return VideoMetadata{}, fmt.Errorf("height parse error: %v", err)
			}
		} else if i == 2 {
			duration = line
		}
	}

	return VideoMetadata{
		Width:    width,
		Height:   height,
		Name:     filepath.Base(inputFilePath),
		Duration: duration,
	}, nil
}

func ListAvailableCodecs() ([]string, error) {
	cmd := exec.Command("ffmpeg", "-codecs")

	var outBuffer bytes.Buffer
	cmd.Stdout = &outBuffer

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("codecs loading error: %v", err)
	}

	codecs := []string{}
	lines := strings.Split(outBuffer.String(), "\n")
	for _, line := range lines {
		if strings.Contains(line, "V") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				codecs = append(codecs, parts[1])
			}
		}
	}

	return codecs, nil
}

func SelectH264Encoder() (string, error) {
	cmd := exec.Command("ffmpeg", "-encoders")

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	encoders := out.String()
	if strings.Contains(encoders, "libx264") {
		return "libx264", nil
	} else if strings.Contains(encoders, "h264_nvenc") {
		return "h264_nvenc", nil
	} else if strings.Contains(encoders, "h264_qsv") {
		return "h264_qsv", nil
	}

	return "", fmt.Errorf("H264 unavailable")
}

func IsValidResolution(resolution string) bool {
	for _, res := range []string{"360p", "480p", "720p", "1080p"} {
		if res == resolution {
			return true
		}
	}

	return false
}

func ProcessVideo(inputFilePath string, videoId string, storages []FileStorage) (*VideoUploadResponse, error) {
	resolutions := []string{"360p", "480p", "720p", "1080p"}
	processedResolutions := make([]Resolution, 0)

	encoder, err := SelectH264Encoder()

	if err != nil {
		return nil, fmt.Errorf("H264 loading encoder error: %v", err)
	}

	outputDir := videoId
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("dir error output creating: %v", err)
	}

	for _, res := range resolutions {
		outputPattern := filepath.Join(outputDir, fmt.Sprintf("video_%s_%%03d.ts", res))
		cmd := exec.Command(
			"ffmpeg",
			"-i", inputFilePath,
			"-vf", fmt.Sprintf("scale=-2:%s", res[:len(res)-1]),
			"-c:v", encoder,
			"-c:a", "aac",
			"-f", "segment",
			"-segment_time", "10",
			"-reset_timestamps", "1",
			"-map", "0",
			outputPattern,
		)

		var errBuffer bytes.Buffer
		cmd.Stderr = &errBuffer

		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("FFMPEG starting error: %v", err)
		}

		if err := cmd.Wait(); err != nil {
			return nil, fmt.Errorf("FFMPEG finishing wait step error: %v, details: %s", err, errBuffer.String())
		}

		segmentIndex := 0
		for {
			segmentFileName := fmt.Sprintf("%s/%s", videoId, VideoSegmentName(res, segmentIndex))
			if _, err := os.Stat(segmentFileName); err != nil {
				if os.IsNotExist(err) {
					break
				}
				return nil, fmt.Errorf("segment error: %v", err)
			}

			segmentBuffer, err := os.ReadFile(segmentFileName)

			if err != nil {
				return nil, fmt.Errorf("buffer reading error %s: %v", segmentFileName, err)
			}

			for _, storage := range storages {
				err := storage.Store(segmentFileName, segmentBuffer)
				if err != nil {
					return nil, fmt.Errorf("buffer store error %s: %v", segmentFileName, err)
				}
			}

			segmentIndex++
		}

		processedResolutions = append(processedResolutions, Resolution{
			Resolution:    res,
			Manifest:      ManifestName(videoId, res),
			TotalSegments: segmentIndex,
		})
	}

	if err := os.RemoveAll(outputDir); err != nil {
		log.Errorf("Error cleaning up output directory: %v", err)
	}

	if err := os.Remove(inputFilePath); err != nil {
		log.Errorf("Error cleaning up input file: %v", err)
	}

	return &VideoUploadResponse{
		Resolutions: processedResolutions,
	}, nil
}

func VideoSegmentName(resolution string, segment int) string {
	return fmt.Sprintf("video_%s_%03d.ts", resolution, segment)
}

func ManifestName(videoUUID string, resolution string) string {
	return fmt.Sprintf("%s/manifest_%s.m3u8", videoUUID, resolution)
}

func GenerateSegmentedManifestSigned(ctx context.Context, videoID string, resolution Resolution, storage FileStorage) (string, error) {
	manifest := "#EXTM3U\n#EXT-X-VERSION:3\n"

	manifest += "#EXT-X-TARGETDURATION:10\n"
	manifest += "#EXT-X-MEDIA-SEQUENCE:0\n"

	for i := 0; i <= resolution.TotalSegments; i++ {
		manifest += "#EXTINF:10.0,\n"
		segmentToSign := fmt.Sprintf("%s/%s", videoID, VideoSegmentName(resolution.Resolution, i))

		signedSegment, err := storage.SignedURL(segmentToSign)

		if err != nil {
			return "", err
		}

		manifest += fmt.Sprintf("%s\n", signedSegment)
	}

	manifest += "#EXT-X-ENDLIST\n"

	manifestPath := ManifestName(videoID, resolution.Resolution)
	err := storage.Store(manifestPath, []byte(manifest))

	if err != nil {
		return "", nil
	}

	manifestSigned, err := storage.SignedURL(manifestPath)

	if err != nil {
		return "", nil
	}

	return manifestSigned, nil
}
