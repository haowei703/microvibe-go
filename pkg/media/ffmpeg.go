package media

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type FFprobeOutput struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Duration  string `json:"duration"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// GetVideoMetadata extracts metadata using ffprobe
func GetVideoMetadata(videoPath string) (duration int, width int, height int, err error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "stream=width,height,duration,codec_type", "-show_entries", "format=duration", "-of", "json", videoPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, 0, 0, fmt.Errorf("ffprobe error: %w", err)
	}

	var probe FFprobeOutput
	if err := json.Unmarshal(out.Bytes(), &probe); err != nil {
		return 0, 0, 0, fmt.Errorf("parse ffprobe error: %w", err)
	}

	for _, s := range probe.Streams {
		if s.CodecType == "video" {
			width = s.Width
			height = s.Height
			if s.Duration != "" {
				if d, pErr := strconv.ParseFloat(s.Duration, 64); pErr == nil {
					duration = int(math.Round(d))
				}
			}
			break
		}
	}

	if duration == 0 && probe.Format.Duration != "" {
		if d, pErr := strconv.ParseFloat(probe.Format.Duration, 64); pErr == nil {
			duration = int(math.Round(d))
		}
	}

	return duration, width, height, nil
}

// ExtractCover grabs the first frame of the video
func ExtractCover(videoPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", videoPath, "-vframes", "1", "-q:v", "2", outputPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg cover error: %w, output: %s", err, string(out))
	}
	return nil
}

// TranscodeToHLS converts the video to HLS format
func TranscodeToHLS(videoPath, outputDir string) error {
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return err
	}

	m3u8Path := filepath.Join(outputDir, "index.m3u8")

	cmd := exec.Command("ffmpeg", "-y", "-i", videoPath,
		"-c:v", "libx264", "-c:a", "aac",
		"-hls_time", "5",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", filepath.Join(outputDir, "segment_%03d.ts"),
		m3u8Path)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg hls error: %w, output: %s", err, string(out))
	}
	return nil
}
