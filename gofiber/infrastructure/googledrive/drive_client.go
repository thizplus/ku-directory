package googledrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"gofiber-template/pkg/config"
)

// DriveClient handles Google Drive API operations
type DriveClient struct {
	config      *oauth2.Config
	webhookURL  string
	httpClient  *http.Client
}

// DriveFile represents a file/folder from Google Drive
type DriveFile struct {
	ID           string
	Name         string
	MimeType     string
	Size         int64
	Description  string
	ThumbnailURL string
	WebViewURL   string
	ParentID     string
	CreatedTime  time.Time
	ModifiedTime time.Time
}

// DriveFolder represents a folder from Google Drive
type DriveFolder struct {
	ID          string
	Name        string
	Path        string
	ParentID    string
	CreatedTime time.Time
}

// TokenInfo represents OAuth token information
type TokenInfo struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Expiry       time.Time
}

// resourceKeyTransport wraps an http.RoundTripper to add resource key header
type resourceKeyTransport struct {
	base        http.RoundTripper
	folderID    string
	resourceKey string
}

// RoundTrip adds the X-Goog-Drive-Resource-Keys header for older shared folders
func (t *resourceKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.resourceKey != "" && t.folderID != "" {
		req.Header.Set("X-Goog-Drive-Resource-Keys", fmt.Sprintf("%s/%s", t.folderID, t.resourceKey))
	}
	return t.base.RoundTrip(req)
}

// NewDriveClient creates a new Google Drive client
func NewDriveClient(cfg config.GoogleDriveConfig) *DriveClient {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes: []string{
			drive.DriveReadonlyScope,           // Read-only access to files
			drive.DriveMetadataReadonlyScope,   // Read-only access to metadata
		},
		Endpoint: google.Endpoint,
	}

	return &DriveClient{
		config:     oauthConfig,
		webhookURL: cfg.WebhookURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAuthURL generates the OAuth authorization URL
func (c *DriveClient) GetAuthURL(state string) string {
	return c.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeCode exchanges authorization code for tokens
func (c *DriveClient) ExchangeCode(ctx context.Context, code string) (*TokenInfo, error) {
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return &TokenInfo{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}, nil
}

// RefreshToken refreshes the access token using refresh token
func (c *DriveClient) RefreshToken(ctx context.Context, refreshToken string) (*TokenInfo, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := c.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return &TokenInfo{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		TokenType:    newToken.TokenType,
		Expiry:       newToken.Expiry,
	}, nil
}

// GetDriveService creates a Drive service with the given tokens
func (c *DriveClient) GetDriveService(ctx context.Context, accessToken, refreshToken string, expiry time.Time) (*drive.Service, error) {
	return c.GetDriveServiceWithResourceKey(ctx, accessToken, refreshToken, expiry, "", "")
}

// GetDriveServiceWithResourceKey creates a Drive service with optional resource key support
// for older shared folders (pre-2021) that require the X-Goog-Drive-Resource-Keys header
func (c *DriveClient) GetDriveServiceWithResourceKey(ctx context.Context, accessToken, refreshToken string, expiry time.Time, folderID, resourceKey string) (*drive.Service, error) {
	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		Expiry:       expiry,
	}

	client := c.config.Client(ctx, token)

	// Wrap with resourceKey transport if resourceKey is provided
	if resourceKey != "" && folderID != "" {
		client.Transport = &resourceKeyTransport{
			base:        client.Transport,
			folderID:    folderID,
			resourceKey: resourceKey,
		}
	}

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	return srv, nil
}

// ListFolders lists all folders in the given parent folder
func (c *DriveClient) ListFolders(ctx context.Context, srv *drive.Service, parentID string) ([]DriveFolder, error) {
	query := "mimeType='application/vnd.google-apps.folder' and trashed=false"
	if parentID != "" {
		query += fmt.Sprintf(" and '%s' in parents", parentID)
	}

	var folders []DriveFolder
	pageToken := ""

	for {
		call := srv.Files.List().
			Q(query).
			Fields("nextPageToken, files(id, name, parents, createdTime)").
			PageSize(100).
			SupportsAllDrives(true).
			IncludeItemsFromAllDrives(true)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		result, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list folders: %w", err)
		}

		for _, f := range result.Files {
			createdTime, _ := time.Parse(time.RFC3339, f.CreatedTime)
			parentID := ""
			if len(f.Parents) > 0 {
				parentID = f.Parents[0]
			}

			folders = append(folders, DriveFolder{
				ID:          f.Id,
				Name:        f.Name,
				ParentID:    parentID,
				CreatedTime: createdTime,
			})
		}

		pageToken = result.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return folders, nil
}

// ListImages lists all image files in the given folder (recursive)
func (c *DriveClient) ListImages(ctx context.Context, srv *drive.Service, folderID string, pageToken string) ([]DriveFile, string, error) {
	// Query for images in the folder
	query := fmt.Sprintf("'%s' in parents and trashed=false and (mimeType contains 'image/')", folderID)

	call := srv.Files.List().
		Q(query).
		Fields("nextPageToken, files(id, name, mimeType, size, description, thumbnailLink, webViewLink, parents, createdTime, modifiedTime, imageMediaMetadata)").
		PageSize(100).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true)

	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	result, err := call.Do()
	if err != nil {
		return nil, "", fmt.Errorf("failed to list images: %w", err)
	}

	var files []DriveFile
	for _, f := range result.Files {
		createdTime, _ := time.Parse(time.RFC3339, f.CreatedTime)
		modifiedTime, _ := time.Parse(time.RFC3339, f.ModifiedTime)
		parentID := ""
		if len(f.Parents) > 0 {
			parentID = f.Parents[0]
		}

		files = append(files, DriveFile{
			ID:           f.Id,
			Name:         f.Name,
			MimeType:     f.MimeType,
			Size:         f.Size,
			Description:  f.Description,
			ThumbnailURL: f.ThumbnailLink,
			WebViewURL:   f.WebViewLink,
			ParentID:     parentID,
			CreatedTime:  createdTime,
			ModifiedTime: modifiedTime,
		})
	}

	return files, result.NextPageToken, nil
}

// ListAllImagesRecursive lists all images in a folder and its subfolders
func (c *DriveClient) ListAllImagesRecursive(ctx context.Context, srv *drive.Service, folderID string) ([]DriveFile, error) {
	var allFiles []DriveFile

	// Get images in current folder
	pageToken := ""
	for {
		files, nextToken, err := c.ListImages(ctx, srv, folderID, pageToken)
		if err != nil {
			return nil, err
		}
		allFiles = append(allFiles, files...)
		pageToken = nextToken
		if pageToken == "" {
			break
		}
	}

	// Get subfolders
	subfolders, err := c.ListFolders(ctx, srv, folderID)
	if err != nil {
		return nil, err
	}

	// Recursively get images from subfolders
	for _, folder := range subfolders {
		subFiles, err := c.ListAllImagesRecursive(ctx, srv, folder.ID)
		if err != nil {
			return nil, err
		}
		allFiles = append(allFiles, subFiles...)
	}

	return allFiles, nil
}

// GetFile gets a single file's metadata
func (c *DriveClient) GetFile(ctx context.Context, srv *drive.Service, fileID string) (*DriveFile, error) {
	f, err := srv.Files.Get(fileID).
		Fields("id, name, mimeType, size, description, thumbnailLink, webViewLink, parents, createdTime, modifiedTime").
		SupportsAllDrives(true).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	createdTime, _ := time.Parse(time.RFC3339, f.CreatedTime)
	modifiedTime, _ := time.Parse(time.RFC3339, f.ModifiedTime)
	parentID := ""
	if len(f.Parents) > 0 {
		parentID = f.Parents[0]
	}

	return &DriveFile{
		ID:           f.Id,
		Name:         f.Name,
		MimeType:     f.MimeType,
		Size:         f.Size,
		Description:  f.Description,
		ThumbnailURL: f.ThumbnailLink,
		WebViewURL:   f.WebViewLink,
		ParentID:     parentID,
		CreatedTime:  createdTime,
		ModifiedTime: modifiedTime,
	}, nil
}

// DownloadFile downloads a file's content
func (c *DriveClient) DownloadFile(ctx context.Context, srv *drive.Service, fileID string) (io.ReadCloser, error) {
	resp, err := srv.Files.Get(fileID).Download()
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	return resp.Body, nil
}

// DownloadThumbnail downloads a file's thumbnail using authenticated HTTP client
func (c *DriveClient) DownloadThumbnail(ctx context.Context, accessToken, refreshToken string, expiry time.Time, fileID string, size int) ([]byte, string, error) {
	// Create authenticated HTTP client
	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		Expiry:       expiry,
	}
	httpClient := c.config.Client(ctx, token)

	// Get Drive service to fetch file metadata
	srv, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create drive service: %w", err)
	}

	// Get file metadata with thumbnail link
	file, err := srv.Files.Get(fileID).
		Fields("id, thumbnailLink, mimeType").
		Do()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get file metadata: %w", err)
	}

	if file.ThumbnailLink == "" {
		return nil, "", fmt.Errorf("no thumbnail available for file %s", fileID)
	}

	// Modify thumbnail URL to get desired resolution
	thumbnailURL := file.ThumbnailLink
	if size > 0 {
		thumbnailURL = strings.Replace(thumbnailURL, "=s220", fmt.Sprintf("=s%d", size), 1)
	}

	// Fetch thumbnail with authenticated client
	resp, err := httpClient.Get(thumbnailURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch thumbnail: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("failed to fetch thumbnail: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read thumbnail: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	return data, contentType, nil
}

// GetFileDownloadURL returns a direct download URL for a file
// This URL can be used by external services to download the file
func (c *DriveClient) GetFileDownloadURL(ctx context.Context, srv *drive.Service, fileID string) (string, error) {
	// Get file metadata with webContentLink
	file, err := srv.Files.Get(fileID).
		Fields("id, webContentLink, thumbnailLink").
		Do()
	if err != nil {
		return "", fmt.Errorf("failed to get file: %w", err)
	}

	// webContentLink is the direct download URL (requires auth for private files)
	if file.WebContentLink != "" {
		return file.WebContentLink, nil
	}

	// Fallback to thumbnail link with higher resolution
	if file.ThumbnailLink != "" {
		// Modify thumbnail URL to get higher resolution (s1600 = 1600px)
		return strings.Replace(file.ThumbnailLink, "=s220", "=s1600", 1), nil
	}

	return "", fmt.Errorf("no download URL available for file %s", fileID)
}

// GetFolderPath gets the full path of a folder
func (c *DriveClient) GetFolderPath(ctx context.Context, srv *drive.Service, folderID string) (string, error) {
	var pathParts []string

	currentID := folderID
	for currentID != "" {
		folder, err := srv.Files.Get(currentID).Fields("id, name, parents").SupportsAllDrives(true).Do()
		if err != nil {
			return "", fmt.Errorf("failed to get folder: %w", err)
		}

		pathParts = append([]string{folder.Name}, pathParts...)

		if len(folder.Parents) > 0 {
			currentID = folder.Parents[0]
		} else {
			break
		}
	}

	return strings.Join(pathParts, "/"), nil
}

// ListAllFoldersRecursive lists all folders recursively starting from a root folder
// This is more efficient than calling GetFolderPath for each file
func (c *DriveClient) ListAllFoldersRecursive(ctx context.Context, srv *drive.Service, rootFolderID string) ([]DriveFolder, error) {
	var allFolders []DriveFolder

	// Get root folder info first
	rootFolder, err := srv.Files.Get(rootFolderID).Fields("id, name, parents").SupportsAllDrives(true).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get root folder: %w", err)
	}

	rootParentID := ""
	if len(rootFolder.Parents) > 0 {
		rootParentID = rootFolder.Parents[0]
	}

	allFolders = append(allFolders, DriveFolder{
		ID:       rootFolder.Id,
		Name:     rootFolder.Name,
		ParentID: rootParentID,
	})

	// Recursively list all subfolders
	var listFoldersRecursive func(parentID string) error
	listFoldersRecursive = func(parentID string) error {
		folders, err := c.ListFolders(ctx, srv, parentID)
		if err != nil {
			return err
		}

		allFolders = append(allFolders, folders...)

		for _, folder := range folders {
			if err := listFoldersRecursive(folder.ID); err != nil {
				return err
			}
		}
		return nil
	}

	if err := listFoldersRecursive(rootFolderID); err != nil {
		return nil, err
	}

	return allFolders, nil
}

// BuildFolderPathMap builds a map of folderID -> full path from a list of folders
// This allows O(1) lookup for folder paths without API calls
func (c *DriveClient) BuildFolderPathMap(folders []DriveFolder, rootFolderID string) map[string]string {
	// Build a map of folderID -> folder for quick lookup
	folderMap := make(map[string]DriveFolder)
	for _, f := range folders {
		folderMap[f.ID] = f
	}

	// Build path map
	pathMap := make(map[string]string)

	var buildPath func(folderID string) string
	buildPath = func(folderID string) string {
		// Check cache first
		if path, exists := pathMap[folderID]; exists {
			return path
		}

		folder, exists := folderMap[folderID]
		if !exists {
			return ""
		}

		// If this is the root folder or has no parent in our map, return just the name
		if folderID == rootFolderID || folder.ParentID == "" {
			pathMap[folderID] = folder.Name
			return folder.Name
		}

		// Check if parent is in our folder map
		_, parentExists := folderMap[folder.ParentID]
		if !parentExists {
			// Parent is outside our root folder, start path from here
			pathMap[folderID] = folder.Name
			return folder.Name
		}

		// Build parent path recursively
		parentPath := buildPath(folder.ParentID)
		if parentPath == "" {
			pathMap[folderID] = folder.Name
		} else {
			pathMap[folderID] = parentPath + "/" + folder.Name
		}

		return pathMap[folderID]
	}

	// Build paths for all folders
	for _, folder := range folders {
		buildPath(folder.ID)
	}

	return pathMap
}

// WatchFolder sets up a webhook to watch for changes in a folder
// NOTE: This uses Files.Watch which only watches the folder itself, not files inside
// Use WatchChanges instead for watching file changes
func (c *DriveClient) WatchFolder(ctx context.Context, srv *drive.Service, folderID, channelID, webhookToken string) (*drive.Channel, error) {
	if c.webhookURL == "" {
		return nil, fmt.Errorf("webhook URL not configured")
	}

	channel := &drive.Channel{
		Id:      channelID,
		Type:    "web_hook",
		Address: c.webhookURL,
		Token:   webhookToken,
		Expiration: time.Now().Add(7 * 24 * time.Hour).UnixMilli(), // 7 days
	}

	// Watch for changes in the folder
	result, err := srv.Files.Watch(folderID, channel).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to watch folder: %w", err)
	}

	return result, nil
}

// WatchChanges sets up a webhook to watch for ALL changes in the user's Drive
// This is the correct way to watch for file additions/deletions in a folder
func (c *DriveClient) WatchChanges(ctx context.Context, srv *drive.Service, channelID, webhookToken, startPageToken string) (*drive.Channel, error) {
	if c.webhookURL == "" {
		return nil, fmt.Errorf("webhook URL not configured")
	}

	channel := &drive.Channel{
		Id:         channelID,
		Type:       "web_hook",
		Address:    c.webhookURL,
		Token:      webhookToken,
		Expiration: time.Now().Add(7 * 24 * time.Hour).UnixMilli(), // 7 days
	}

	// Watch for ALL changes in the user's Drive
	result, err := srv.Changes.Watch(startPageToken, channel).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to watch changes: %w", err)
	}

	return result, nil
}

// StopWatch stops watching a channel
func (c *DriveClient) StopWatch(ctx context.Context, srv *drive.Service, channelID, resourceID string) error {
	channel := &drive.Channel{
		Id:         channelID,
		ResourceId: resourceID,
	}

	return srv.Channels.Stop(channel).Do()
}

// GetChanges gets changes since the given start page token
func (c *DriveClient) GetChanges(ctx context.Context, srv *drive.Service, startPageToken string) ([]*drive.Change, string, error) {
	var changes []*drive.Change
	pageToken := startPageToken

	for {
		result, err := srv.Changes.List(pageToken).
			Fields("nextPageToken, newStartPageToken, changes(fileId, file, removed, time)").
			PageSize(100).
			SupportsAllDrives(true).
			IncludeItemsFromAllDrives(true).
			Do()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get changes: %w", err)
		}

		changes = append(changes, result.Changes...)

		if result.NewStartPageToken != "" {
			return changes, result.NewStartPageToken, nil
		}

		pageToken = result.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return changes, "", nil
}

// GetStartPageToken gets the start page token for change tracking
func (c *DriveClient) GetStartPageToken(ctx context.Context, srv *drive.Service) (string, error) {
	token, err := srv.Changes.GetStartPageToken().Do()
	if err != nil {
		return "", fmt.Errorf("failed to get start page token: %w", err)
	}
	return token.StartPageToken, nil
}

// ValidateConfig checks if the configuration is valid
func (c *DriveClient) ValidateConfig() error {
	if c.config.ClientID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID is not configured")
	}
	if c.config.ClientSecret == "" {
		return fmt.Errorf("GOOGLE_CLIENT_SECRET is not configured")
	}
	return nil
}

// ParseWebhookHeaders parses webhook headers from Google Drive
type WebhookPayload struct {
	ChannelID         string
	ResourceID        string
	ResourceState     string
	ResourceURI       string
	ChannelExpiration time.Time
	ChannelToken      string
}

func ParseWebhookHeaders(headers http.Header) *WebhookPayload {
	expiration, _ := time.Parse(time.RFC3339, headers.Get("X-Goog-Channel-Expiration"))

	return &WebhookPayload{
		ChannelID:         headers.Get("X-Goog-Channel-Id"),
		ResourceID:        headers.Get("X-Goog-Resource-Id"),
		ResourceState:     headers.Get("X-Goog-Resource-State"),
		ResourceURI:       headers.Get("X-Goog-Resource-Uri"),
		ChannelExpiration: expiration,
		ChannelToken:      headers.Get("X-Goog-Channel-Token"),
	}
}

// ToJSON converts TokenInfo to JSON string
func (t *TokenInfo) ToJSON() (string, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// TokenInfoFromJSON creates TokenInfo from JSON string
func TokenInfoFromJSON(data string) (*TokenInfo, error) {
	var t TokenInfo
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return nil, err
	}
	return &t, nil
}
