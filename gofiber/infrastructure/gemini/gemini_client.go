package gemini

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
)

// GeminiClient wraps the Google Gemini API client
type GeminiClient struct {
	client *genai.Client
	model  string
}

// NewsArticle represents the generated news article
type NewsArticle struct {
	Title      string      `json:"title"`
	Paragraphs []Paragraph `json:"paragraphs"`
	Tags       []string    `json:"tags"`
}

// Paragraph represents a single paragraph in the article
type Paragraph struct {
	Heading string `json:"heading"`
	Content string `json:"content"`
}

// GenerateNewsRequest contains the parameters for news generation
type GenerateNewsRequest struct {
	FolderName string   // Activity/folder name for context
	Images     [][]byte // Image data
	MimeTypes  []string // MIME types for each image
	Headings   []string // Custom headings for 4 paragraphs
	Tone       string   // formal, friendly, news
	Length     string   // short, medium, long
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(apiKey, model string) (*GeminiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiClient{
		client: client,
		model:  model,
	}, nil
}

// GenerateNews generates a news article from images or text-only
func (c *GeminiClient) GenerateNews(ctx context.Context, req *GenerateNewsRequest) (*NewsArticle, error) {
	// Build the prompt (different based on whether images are provided)
	prompt := buildNewsPrompt(req)

	// Build parts with images and text
	var parts []*genai.Part

	// Add images first (if any)
	for i, imgData := range req.Images {
		mimeType := "image/jpeg"
		if i < len(req.MimeTypes) && req.MimeTypes[i] != "" {
			mimeType = req.MimeTypes[i]
		}
		parts = append(parts, genai.NewPartFromBytes(imgData, mimeType))
	}

	// Add text prompt
	parts = append(parts, genai.NewPartFromText(prompt))

	// Create content
	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	// Configure for JSON output
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"title": {
					Type:        genai.TypeString,
					Description: "หัวข้อข่าว",
				},
				"paragraphs": {
					Type: genai.TypeArray,
					Items: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"heading": {Type: genai.TypeString, Description: "หัวข้อย่อหน้า"},
							"content": {Type: genai.TypeString, Description: "เนื้อหาย่อหน้า"},
						},
						Required: []string{"heading", "content"},
					},
					Description: "4 ย่อหน้าของข่าว",
				},
				"tags": {
					Type:        genai.TypeArray,
					Items:       &genai.Schema{Type: genai.TypeString},
					Description: "แท็กที่เกี่ยวข้อง",
				},
			},
			Required: []string{"title", "paragraphs", "tags"},
		},
	}

	// Generate content
	result, err := c.client.Models.GenerateContent(ctx, c.model, contents, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// Parse response
	if len(result.Candidates) == 0 || result.Candidates[0].Content == nil {
		return nil, fmt.Errorf("no content generated")
	}

	// Get text from response
	text := result.Text()
	if text == "" {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	// Parse JSON
	var article NewsArticle
	if err := json.Unmarshal([]byte(text), &article); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &article, nil
}

// buildNewsPrompt builds the prompt for news generation
func buildNewsPrompt(req *GenerateNewsRequest) string {
	// Determine word count based on length
	wordCount := "200"
	switch req.Length {
	case "short":
		wordCount = "100"
	case "long":
		wordCount = "300+"
	}

	// Determine tone description
	toneDesc := "เป็นทางการ"
	switch req.Tone {
	case "friendly":
		toneDesc = "เป็นกันเอง อบอุ่น"
	case "news":
		toneDesc = "สไตล์ข่าว กระชับ ตรงประเด็น"
	}

	// Build headings instruction
	headingsText := ""
	if len(req.Headings) == 4 {
		headingsText = fmt.Sprintf(`
โครงสร้าง 4 ย่อหน้าที่ต้องการ:
1. %s
2. %s
3. %s
4. %s
`, req.Headings[0], req.Headings[1], req.Headings[2], req.Headings[3])
	} else {
		headingsText = `
โครงสร้าง 4 ย่อหน้าที่ต้องการ:
1. ความเป็นมา - บอกที่มาและวัตถุประสงค์ของกิจกรรม
2. กิจกรรมที่จัด - อธิบายรายละเอียดกิจกรรมที่เกิดขึ้น
3. ผู้เข้าร่วม - บอกถึงผู้เข้าร่วมและบรรยากาศ
4. สรุป - สรุปผลและความสำเร็จของกิจกรรม
`
	}

	// Build different prompts based on whether images are provided
	var prompt string
	if len(req.Images) > 0 {
		// Prompt with images
		prompt = fmt.Sprintf(`คุณเป็นนักเขียนข่าวประชาสัมพันธ์มืออาชีพ

กิจกรรม: %s

วิเคราะห์รูปภาพเหล่านี้และเขียนข่าวประชาสัมพันธ์ภาษาไทย
%s
ข้อกำหนด:
- โทนการเขียน: %s
- ความยาวรวม: ประมาณ %s คำ
- ใช้ภาษาไทยที่สละสลวย เข้าใจง่าย
- หัวข้อข่าวต้องดึงดูดความสนใจ
- แท็กควรเป็นคำสำคัญ 3-5 คำ

ตอบเป็น JSON ตามโครงสร้างที่กำหนด`, req.FolderName, headingsText, toneDesc, wordCount)
	} else {
		// Prompt without images (text-only based on headings)
		prompt = fmt.Sprintf(`คุณเป็นนักเขียนข่าวประชาสัมพันธ์มืออาชีพ

เขียนข่าวประชาสัมพันธ์ภาษาไทยตามหัวข้อที่กำหนด
%s
ข้อกำหนด:
- โทนการเขียน: %s
- ความยาวรวม: ประมาณ %s คำ
- ใช้ภาษาไทยที่สละสลวย เข้าใจง่าย
- หัวข้อข่าวต้องดึงดูดความสนใจ
- เขียนเนื้อหาตามหัวข้อย่อหน้าที่กำหนดให้ครบถ้วน
- แท็กควรเป็นคำสำคัญ 3-5 คำ

ตอบเป็น JSON ตามโครงสร้างที่กำหนด`, headingsText, toneDesc, wordCount)
	}

	return prompt
}

// Close closes the Gemini client
func (c *GeminiClient) Close() error {
	// The genai client doesn't have a Close method in the current SDK
	return nil
}
