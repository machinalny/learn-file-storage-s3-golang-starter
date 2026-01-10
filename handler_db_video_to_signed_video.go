package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}

	if strings.Contains(*video.VideoURL, "https") {
		return video, nil
	}
	fmt.Println(*video.VideoURL)
	parts := strings.Split(*video.VideoURL, ",")
	bucket := parts[0]
	key := parts[1]

	presignedUrl, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Duration(3600)*time.Second)
	if err != nil {
		return video, err
	}
	video.VideoURL = &presignedUrl
	return video, nil
}
