package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	maxMemory := 1 << 30
	err = r.ParseMultipartForm(int64(maxMemory))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Video is too big", err)
		return
	}

	videoFile, imageHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read video file", err)
		return
	}
	defer videoFile.Close()

	videoMetadata, err := cfg.db.GetVideo(videoID)

	if err != nil || videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not video owner", err)
		return
	}

	mimeType, _, err := mime.ParseMediaType(imageHeader.Header.Get("Content-Type"))
	if err != nil || (mimeType != "video/mp4" && mimeType != "image/png") {
		if err == nil {
			err = errors.New("Unsupported media type")
		}
		respondWithError(w, http.StatusUnsupportedMediaType, "Only mp4 is supported", err)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload*.mp4")
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save file", err)
		return
	}

	_, err = io.Copy(tempFile, videoFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save video file", err)
		return
	}

	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to retrieve aspect ratio", err)
		return
	}

	aspectRatioPrefix := "other"
	switch aspectRatio {
	case "9:16":
		aspectRatioPrefix = "portrait"
	case "16:9":
		aspectRatioPrefix = "landscape"
	}

	randSlice := [32]byte{}
	_, _ = rand.Read(randSlice[:])
	randomFileName := aspectRatioPrefix + "/" + base64.RawURLEncoding.EncodeToString(randSlice[:]) + ".mp4"

	tempFile.Seek(0, io.SeekStart)
	cfg.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &randomFileName,
		Body:        tempFile,
		ContentType: &mimeType,
	})

	videoUrl := fmt.Sprintf("http://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, randomFileName)
	videoMetadata.VideoURL = &videoUrl

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Video update issue", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
