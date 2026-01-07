# File Upload Examples

The Go Fiber file upload system now supports flexible custom path configurations. You can use either a structured approach or specify custom paths directly.

## Upload Approaches

### 1. Structured Path Approach (Default)

Uses organized folder structure based on category, entity ID, and file type.

**Example using curl:**
```bash
curl -X POST "http://localhost:8080/api/v1/files/upload" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "file=@/path/to/your/file.jpg" \
  -F "category=profiles" \
  -F "entity_id=123e4567-e89b-12d3-a456-426614174000" \
  -F "file_type=avatar"
```

**Generated path:** `uploads/{user_id}/profiles/123e4567-e89b-12d3-a456-426614174000/avatar/{unique_filename}.jpg`

### 2. Custom Path Approach

Specify exact upload path (with validation for security).

**Example using curl:**
```bash
curl -X POST "http://localhost:8080/api/v1/files/upload" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "file=@/path/to/your/file.jpg" \
  -F "custom_path=my-custom-folder/subfolder"
```

**Generated path:** `my-custom-folder/subfolder/{unique_filename}.jpg`

## Form Data Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `file` | File | Yes | The file to upload (filename and MIME type auto-extracted) |
| `custom_path` | String | No | Custom path for file storage (max 500 chars) |
| `category` | String | No | Category for structured path (max 50 chars) |
| `entity_id` | String | No | Entity UUID for structured path |
| `file_type` | String | No | File type for structured path (max 50 chars) |

**Note:** The filename, MIME type, and file size are automatically extracted from the uploaded file. You do not need to provide these values separately.

## Path Validation Rules

### Custom Path Security
- No directory traversal (`../` patterns)
- No absolute paths (`/` or `C:\` patterns)
- No dangerous characters (`<>:"|?*` and control chars)
- No Windows reserved names (`CON`, `PRN`, `AUX`, etc.)
- Maximum length: 500 characters

### Filename Sanitization
- Removes dangerous characters
- Prevents path injection
- Generates unique filenames to prevent conflicts

## Response Format

```json
{
  "success": true,
  "message": "File uploaded successfully",
  "data": {
    "fileId": "uuid",
    "fileName": "sanitized_filename.jpg",
    "url": "https://cdn.example.com/path/to/file.jpg",
    "cdnPath": "actual/storage/path/filename.jpg",
    "fileSize": 12345,
    "mimeType": "image/jpeg",
    "pathType": "custom" // or "structured"
  }
}
```

## Error Handling

### Validation Errors
```json
{
  "success": false,
  "message": "Validation failed",
  "errors": {
    "customPath": ["path contains invalid characters"]
  }
}
```

### Path Security Errors
```json
{
  "success": false,
  "message": "File upload failed",
  "error": "invalid custom path: unsafe path detected"
}
```

## Usage Examples in Different Scenarios

### 1. User Profile Pictures
```bash
# Structured approach
-F "category=users" -F "entity_id={user_id}" -F "file_type=avatar"

# Custom approach
-F "custom_path=user-avatars"
```

### 2. Document Storage
```bash
# Structured approach
-F "category=documents" -F "entity_id={project_id}" -F "file_type=contracts"

# Custom approach
-F "custom_path=legal/contracts/2024"
```

### 3. Temporary Files
```bash
# Custom approach for temp files
-F "custom_path=temp/uploads"
```

## Best Practices

1. **Use structured approach** for organized, searchable file systems
2. **Use custom paths** for specific requirements or legacy compatibility
3. **Always validate paths** on the client side before sending
4. **Monitor storage usage** and implement cleanup for temporary files
5. **Use appropriate file type restrictions** based on your use case