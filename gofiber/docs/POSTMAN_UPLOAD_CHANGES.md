# Postman Collection Updates - File Upload Enhancement

## Overview

Updated the `GoFiber-Starter-API.postman_collection.json` to support the new flexible file upload system with both structured and custom path approaches.

## Changes Made

### 1. Collection Information Updates
- **Version**: Updated from 1.0.0 to 1.1.0
- **Description**: Enhanced to highlight new file upload features
- **Features**: Added documentation for structured paths, custom paths, and security validation

### 2. File Upload Requests - Complete Overhaul

#### A. Upload File - Structured Path (NEW)
**Original**: Single basic upload request
**New**: Dedicated structured path upload with comprehensive testing

**Form Data Fields**:
- `file` (required): The file to upload
- `category` (optional): Category for organization (e.g., 'profiles', 'tasks')
- `entity_id` (optional): Related entity UUID (e.g., user ID, task ID)
- `file_type` (optional): File type/purpose (e.g., 'avatar', 'documents')

**Test Validations**:
- Status code verification (200)
- Response structure validation
- Path type confirmation ('structured')
- Path structure verification (contains expected elements)
- File ID saving for subsequent tests

#### B. Upload File - Custom Path (NEW)
**Purpose**: Test custom path upload approach

**Form Data Fields**:
- `file` (required): The file to upload
- `custom_path` (required): Custom directory path

**Test Validations**:
- Status code verification (200)
- Response structure validation
- Path type confirmation ('custom')
- Custom path application verification
- File ID saving for cleanup

#### C. Upload File - Invalid Custom Path (NEW)
**Purpose**: Security testing with malicious paths

**Form Data Fields**:
- `file` (required): Test file
- `custom_path`: Malicious path (`../../../etc/passwd`)

**Test Validations**:
- Status code verification (400)
- Error response validation
- Security error message checking

#### D. Upload File - Multiple Path Security Tests (NEW)
**Purpose**: Comprehensive security testing with dynamic path selection

**Features**:
- **Pre-request Script**: Dynamically selects from various malicious paths
- **Test Paths**: Directory traversal, absolute paths, script injection, reserved names, long paths, special characters
- **Dynamic Testing**: Random selection for comprehensive coverage

**Test Validations**:
- Status code verification (400)
- Malicious path rejection
- Appropriate security error messages
- Performance validation (response time < 2000ms)

#### E. Upload File - Task Documents (Structured) (NEW)
**Purpose**: Real-world example of structured path for task management

**Form Data Fields**:
- `file`: Task document
- `category`: "tasks"
- `entity_id`: "{{task_id}}"
- `file_type`: "documents"

**Generated Path**: `uploads/{user_id}/tasks/{task_id}/documents/{filename}`

#### F. Upload File - Temporary Storage (Custom) (NEW)
**Purpose**: Real-world example of custom path for temporary storage

**Form Data Fields**:
- `file`: Temporary file
- `custom_path`: "temp/uploads"

**Generated Path**: `temp/uploads/{filename}`

### 3. Enhanced Testing & Validation

#### Response Structure Validation
All upload requests now validate:
- `fileId`: File identifier
- `fileName`: Sanitized filename
- `url`: CDN URL
- `cdnPath`: Storage path
- `fileSize`: File size
- `mimeType`: Content type
- `pathType`: Upload approach used ('structured' or 'custom')

#### Path Validation Testing
- **Structured paths**: Verify organized folder structure
- **Custom paths**: Verify exact path application
- **Security paths**: Verify rejection of malicious attempts

#### Environment Variables
Added new environment variables for testing:
- `test_custom_path`: Valid custom path for testing
- `test_category`: Default category for structured uploads
- `test_file_type`: Default file type for structured uploads
- `invalid_path`: Directory traversal attempt
- `malicious_path`: Script injection attempt
- `current_malicious_path`: Dynamically set malicious path

### 4. Documentation Enhancements

#### Request Descriptions
Each upload request includes:
- **Purpose**: Clear explanation of the upload approach
- **Use Cases**: Real-world scenarios
- **Form Data Fields**: Detailed parameter descriptions
- **Generated Path**: Expected path structure
- **Security Features**: Validation mechanisms
- **Benefits**: Advantages of each approach

#### Security Documentation
- **Path Validation Rules**: Comprehensive security measures
- **Valid Examples**: Safe path patterns
- **Invalid Examples**: Blocked path patterns
- **Security Features**: Protection mechanisms

### 5. Files Section Description Update

Enhanced the Files folder description to include:
- **Upload Approaches**: Overview of structured vs custom paths
- **Path Examples**: Concrete examples of each approach
- **Security Testing**: Information about validation tests

## Testing Scenarios Covered

### 1. Functional Testing
- ✅ Structured path uploads (profiles, tasks, documents)
- ✅ Custom path uploads (temporary storage, custom folders)
- ✅ Response validation and file ID extraction
- ✅ Path type identification

### 2. Security Testing
- ✅ Directory traversal prevention (`../../../etc/passwd`)
- ✅ Absolute path blocking (`/absolute/path`)
- ✅ Script injection protection (`<script>alert(1)</script>`)
- ✅ Windows reserved name validation (`CON/file.txt`)
- ✅ Path length validation (>500 characters)
- ✅ Special character filtering (`|`, `<`, `>`, etc.)

### 3. Integration Testing
- ✅ Authentication token usage
- ✅ User context integration
- ✅ File cleanup preparation
- ✅ Cross-request data sharing (file IDs)

### 4. Performance Testing
- ✅ Response time validation (<2000ms for security tests)
- ✅ Large file handling preparation
- ✅ Concurrent upload readiness

## Usage Instructions

### For Structured Uploads
1. Select "Upload File - Structured Path"
2. Set appropriate `category`, `entity_id`, and `file_type`
3. Choose file to upload
4. Verify organized path structure in response

### For Custom Uploads
1. Select "Upload File - Custom Path"
2. Set desired `custom_path`
3. Choose file to upload
4. Verify custom path application in response

### For Security Testing
1. Run "Upload File - Invalid Custom Path" for basic security test
2. Run "Upload File - Multiple Path Security Tests" for comprehensive testing
3. Verify all malicious paths are properly rejected

### For Real-World Examples
1. Use "Upload File - Task Documents" for task management scenarios
2. Use "Upload File - Temporary Storage" for temporary file handling

## Environment Setup

Ensure these environment variables are set:
- `base_url`: API base URL
- `auth_token`: Valid JWT token (auto-set from login)
- `user_id`: Current user ID (auto-set from login)
- `task_id`: Valid task ID (auto-set from task creation)

## Benefits of the Updated Collection

1. **Comprehensive Coverage**: Tests both upload approaches thoroughly
2. **Security Validation**: Robust testing of path validation mechanisms
3. **Real-World Examples**: Practical usage scenarios for different needs
4. **Dynamic Testing**: Randomized security tests for better coverage
5. **Clear Documentation**: Detailed descriptions for each approach
6. **Environment Integration**: Seamless integration with existing test data
7. **Scalable Structure**: Easy to add more upload scenarios in the future

## Version History

- **v1.0.0**: Basic file upload functionality
- **v1.1.0**: Enhanced with flexible path support, security validation, and comprehensive testing scenarios