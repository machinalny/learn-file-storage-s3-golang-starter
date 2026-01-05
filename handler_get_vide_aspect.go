package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

type Stream struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type FFProbeOutput struct {
	Streams []Stream `json:"streams"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	// Run ffprobe command to get video metadata
	command := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	buffer := bytes.NewBuffer([]byte{})
	command.Stdout = buffer

	errorBuffer := bytes.NewBuffer([]byte{})
	command.Stderr = errorBuffer

	err := command.Run()
	if err != nil {
		fmt.Println("Error buffer" + errorBuffer.String())
		return "", fmt.Errorf("ffprobe command failed: %w", err)
	}

	var output FFProbeOutput
	err = json.Unmarshal(buffer.Bytes(), &output)
	if err != nil {
		return "", fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	// Find the first video stream
	for _, stream := range output.Streams {
		if stream.Width > 0 && stream.Height > 0 {
			return calculateAspectRatio(stream.Width, stream.Height), nil
		}
	}

	return "", fmt.Errorf("no valid video stream found")
}

// calculateAspectRatio calculates the aspect ratio from width and height
func calculateAspectRatio(width, height int) string {
	if height == 0 {
		return "undefined"
	}

	// Calculate the decimal ratio
	ratio := float64(width) / float64(height)

	// Try to match common aspect ratios
	commonRatios := map[string]float64{
		"16:9": 16.0 / 9.0, // ~1.778
		"9:16": 9.0 / 16.0, // ~0.563 (vertical)
	}

	// Check if the ratio matches any common ratio (with small tolerance)
	tolerance := 0.01
	for ratioName, commonRatio := range commonRatios {
		if math.Abs(ratio-commonRatio) < tolerance {
			return ratioName
		}
	}

	// If no common ratio matches, return the decimal ratio
	return "other"
}
