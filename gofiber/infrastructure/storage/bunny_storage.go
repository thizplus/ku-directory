package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type BunnyStorage interface {
	UploadFile(file io.Reader, path string, contentType string) (string, error)
	DeleteFile(path string) error
	GetFileURL(path string) string
}

type BunnyStorageImpl struct {
	storageZone string
	accessKey   string
	baseURL     string
	cdnURL      string
}

type BunnyConfig struct {
	StorageZone string
	AccessKey   string
	BaseURL     string
	CDNUrl      string
}

func NewBunnyStorage(config BunnyConfig) BunnyStorage {
	return &BunnyStorageImpl{
		storageZone: config.StorageZone,
		accessKey:   config.AccessKey,
		baseURL:     config.BaseURL,
		cdnURL:      config.CDNUrl,
	}
}

func (b *BunnyStorageImpl) UploadFile(file io.Reader, path string, contentType string) (string, error) {
	url := fmt.Sprintf("%s/%s/%s", b.baseURL, b.storageZone, path)

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(fileBytes))
	if err != nil {
		return "", err
	}

	// แก้ไข header name
	req.Header.Set("AccessKey", b.accessKey) // หรือลอง Authorization
	// req.Header.Set("Authorization", "Bearer " + b.accessKey)  // ทางเลือก
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(fileBytes)))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// เพิ่ม debug
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	fileURL := b.GetFileURL(path)
	return fileURL, nil
}

func (b *BunnyStorageImpl) DeleteFile(path string) error {
	url := fmt.Sprintf("%s/%s/%s", b.baseURL, b.storageZone, path)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("AccessKey", b.accessKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (b *BunnyStorageImpl) GetFileURL(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return b.cdnURL + path
}
