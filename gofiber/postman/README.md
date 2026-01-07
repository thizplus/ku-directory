# GoFiber Starter API - Postman Collection

Complete Postman testing suite for the GoFiber Starter Project with comprehensive API coverage, authentication workflows, and environment configurations.

## 📁 Files Included

- `GoFiber-Starter-API.postman_collection.json` - Main collection with all API endpoints
- `Development.postman_environment.json` - Development environment variables
- `Staging.postman_environment.json` - Staging environment variables
- `Production.postman_environment.json` - Production environment variables

## 🚀 Quick Start

### 1. Import Collection & Environment

1. Open Postman
2. Click **Import** button
3. Drag and drop all JSON files or select them manually
4. Select the appropriate environment (Development/Staging/Production)

### 2. Set Environment Variables

Before running tests, update the environment variables:

**Development:**
- `base_url`: `http://localhost:8080` (default)
- `database_url`: Your local PostgreSQL connection
- `redis_url`: Your local Redis connection
- `jwt_secret`: Your JWT secret key

**Staging/Production:**
- Update URLs to match your deployed environments
- Set secure credentials and secrets
- Disable test user variables in production

### 3. Authentication Workflow

1. **Register User** - Creates a new test user account
2. **Login User** - Authenticates and automatically saves JWT token
3. All subsequent requests will use the saved token automatically

## 📋 Collection Structure

### 🔐 Authentication
- **Register User** - Create new user account
- **Login User** - Authenticate and get JWT token
- **Login - Invalid Credentials** - Error scenario testing

### 👤 Users
- **Get User Profile** - Retrieve current user profile
- **Update User Profile** - Modify user information
- **List Users** - Get paginated user list
- **Delete User Account** - Remove user account

### 📋 Tasks
- **Create Task** - Add new task
- **Get Task** - Retrieve specific task
- **Update Task** - Modify task details
- **List User Tasks** - Get current user's tasks
- **List All Tasks** - Get all system tasks
- **Delete Task** - Remove task

### 📁 Files
- **Upload File** - Upload file with multipart form
- **Get File** - Retrieve file metadata
- **List User Files** - Get current user's files
- **List All Files** - Get all system files
- **Delete File** - Remove file and storage

### ⚙️ Jobs
- **Create Job** - Schedule new cron job
- **Get Job** - Retrieve job details
- **Update Job** - Modify job configuration
- **List Jobs** - Get all scheduled jobs
- **Start Job** - Activate job scheduling
- **Stop Job** - Deactivate job scheduling
- **Delete Job** - Remove job permanently

### ❤️ Health
- **Health Check** - API availability check
- **API Info** - Get API version and environment info

## 🧪 Testing Features

### Automated Tests
Each request includes comprehensive test scripts:

- **Status Code Validation** - Verifies correct HTTP responses
- **Response Structure Validation** - Checks JSON structure and required fields
- **Data Type Validation** - Ensures correct data types
- **Business Logic Validation** - Verifies business rules
- **Error Handling Tests** - Validates error responses
- **Performance Tests** - Checks response times

### Authentication Flow
- Automatic token extraction and storage
- Token usage in subsequent requests
- User context preservation across requests

### Dynamic Test Data
- Automatic test user generation with timestamps
- Dynamic ID extraction and reuse
- Environment-specific configurations

## 🔧 Environment Variables

### Runtime Variables (Auto-populated)
- `auth_token` - JWT token from login
- `user_id` - Current user ID
- `user_email` - Current user email
- `task_id` - Last created task ID
- `job_id` - Last created job ID
- `file_id` - Last uploaded file ID

### Test Data Variables (Auto-generated)
- `test_email` - Dynamic test email
- `test_username` - Dynamic username
- `test_password` - Test password
- `test_first_name` - Test first name
- `test_last_name` - Test last name

### Configuration Variables
- `base_url` - API base URL
- `api_version` - API version
- `environment` - Environment name
- `timeout` - Request timeout
- `debug_mode` - Debug flag

## 📝 Sample Test Data

### User Registration
```json
{
    "email": "test1642594800000@example.com",
    "username": "testuser1642594800000",
    "password": "testpassword123",
    "firstName": "Test",
    "lastName": "User"
}
```

### Task Creation
```json
{
    "title": "Sample Task",
    "description": "This is a sample task for testing",
    "priority": 3,
    "dueDate": "2024-12-31T23:59:59Z"
}
```

### Job Scheduling
```json
{
    "name": "Daily Report Job",
    "cronExpr": "0 9 * * *",
    "payload": "{\"reportType\": \"daily\", \"recipients\": [\"admin@example.com\"]}"
}
```

## 🔒 Security Notes

### Development Environment
- Uses localhost URLs
- Debug mode enabled
- Simple JWT secrets (change for production)
- Test credentials included

### Production Environment
- Test variables disabled
- Secure credential storage required
- HTTPS endpoints only
- Strong JWT secrets mandatory

## 🚦 Running Tests

### Manual Testing
1. Select environment
2. Run authentication requests first
3. Execute other endpoints in any order
4. Check test results in Test Results tab

### Collection Runner
1. Click **Runner** button
2. Select collection and environment
3. Configure iterations and data
4. Run complete test suite
5. Review detailed test report

### CI/CD Integration
```bash
# Install Newman (Postman CLI)
npm install -g newman

# Run collection with environment
newman run GoFiber-Starter-API.postman_collection.json \
  -e Development.postman_environment.json \
  --reporters cli,html \
  --reporter-html-export report.html
```

## 🐛 Troubleshooting

### Common Issues

**Authentication Failed**
- Verify user registration completed
- Check JWT token in environment variables
- Ensure token hasn't expired

**Invalid Base URL**
- Confirm API server is running
- Check environment URL configuration
- Verify network connectivity

**File Upload Failed**
- Ensure file path is correct in form data
- Check file size limits
- Verify storage configuration

**Job Creation Failed**
- Validate cron expression syntax
- Check job name uniqueness
- Verify scheduler service is running

### Error Response Format
```json
{
    "success": false,
    "message": "Error description",
    "error": "Detailed error information"
}
```

## 📚 API Documentation

For detailed API documentation, refer to:
- Swagger/OpenAPI documentation (if available)
- API source code in handlers directory
- Individual request descriptions in Postman

## 🤝 Contributing

When adding new endpoints:
1. Add request to appropriate folder
2. Include comprehensive test scripts
3. Update environment variables if needed
4. Add documentation in request description
5. Test across all environments

## 📄 License

This Postman collection is part of the GoFiber Starter Project and follows the same licensing terms.