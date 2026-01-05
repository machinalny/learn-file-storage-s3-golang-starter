package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	// TODO: implement the upload here
	maxMemory := 10 << 20
	err = r.ParseMultipartForm(int64(maxMemory))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Thumbnail is too big", err)
		return
	}

	imageFile, imageHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read thumbnail file", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)

	if err != nil || videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not video owner", err)
		return
	}

	mimeType, _, err := mime.ParseMediaType(imageHeader.Header.Get("Content-Type"))
	if err != nil || (mimeType != "image/jpeg" && mimeType != "image/png") {
		if err == nil {
			err = errors.New("Unsupported media type")
		}
		respondWithError(w, http.StatusUnsupportedMediaType, "Only jpeg and png are supported", err)
		return
	}
	randSlice := [32]byte{}
	_, _ = rand.Read(randSlice[:])
	randomFileName := base64.RawURLEncoding.EncodeToString(randSlice[:])
	fileExtension := strings.Split(imageHeader.Header.Get("Content-Type"), "/")[1]
	uniquePath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", randomFileName, fileExtension))

	newFile, err := os.Create(uniquePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save thumbnail file", err)
		return
	}

	_, err = io.Copy(newFile, imageFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save thumbnail file", err)
		return
	}

	thumbnailUrl := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, randomFileName, fileExtension)
	videoMetadata.ThumbnailURL = &thumbnailUrl

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Video update issue", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
