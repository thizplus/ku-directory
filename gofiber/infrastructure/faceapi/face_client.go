package faceapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FaceClient communicates with the Face Detection Python service
type FaceClient struct {
	baseURL    string
	httpClient *http.Client
}

// DetectedFace represents a detected face from the API
type DetectedFace struct {
	// Bounding box (normalized 0-1)
	BboxX      float64 `json:"bbox_x"`
	BboxY      float64 `json:"bbox_y"`
	BboxWidth  float64 `json:"bbox_width"`
	BboxHeight float64 `json:"bbox_height"`

	// Face embedding (512 dimensions for InsightFace)
	Embedding []float32 `json:"embedding"`

	// Detection confidence
	Confidence float64 `json:"confidence"`
}

// ExtractRequest is the request to extract faces from an image
type ExtractRequest struct {
	ImageURL string `json:"image_url"`
}

// ExtractResponse is the response from face extraction
type ExtractResponse struct {
	Success bool           `json:"success"`
	Faces   []DetectedFace `json:"faces"`
	Error   string         `json:"error,omitempty"`

	// Processing info
	ProcessingTimeMs int `json:"processing_time_ms"`
}

// HealthResponse is the response from health check
type HealthResponse struct {
	Status  string `json:"status"`
	Model   string `json:"model"`
	Version string `json:"version"`
}

// NewFaceClient creates a new face API client
func NewFaceClient(baseURL string) *FaceClient {
	return &FaceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Face processing can take time, especially on CPU
		},
	}
}

// ExtractFaces extracts faces from an image URL
func (c *FaceClient) ExtractFaces(ctx context.Context, imageURL string) (*ExtractResponse, error) {
	reqBody := ExtractRequest{
		ImageURL: imageURL,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/extract", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call face API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("face API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result ExtractResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("face extraction failed: %s", result.Error)
	}

	return &result, nil
}

// ExtractFacesFromBytes extracts faces from image bytes
func (c *FaceClient) ExtractFacesFromBytes(ctx context.Context, imageData []byte, mimeType string) (*ExtractResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/extract-bytes", bytes.NewBuffer(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", mimeType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call face API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("face API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result ExtractResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("face extraction failed: %s", result.Error)
	}

	return &result, nil
}

// Health checks if the face API is healthy
func (c *FaceClient) Health(ctx context.Context) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call health API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	var result HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// IsAvailable checks if the face API is available
func (c *FaceClient) IsAvailable(ctx context.Context) bool {
	health, err := c.Health(ctx)
	if err != nil {
		return false
	}
	return health.Status == "ok"
}
