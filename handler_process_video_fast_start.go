package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

func processVideoForFastStart(filePath string) (string, error) {
	processingFilePah := filePath + ".processing"

	command := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", processingFilePah)
	errBuffer := bytes.NewBuffer([]byte{})
	command.Stderr = errBuffer
	err := command.Run()
	if err != nil {
		fmt.Println("Unable to process video " + errBuffer.String())
		return "", err
	}
	return processingFilePah, nil
}
