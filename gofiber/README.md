# GoFiber Template Project

A complete Go Fiber template project with Clean Architecture featuring:

## Features

- **Clean Architecture** - Domain, Application, Infrastructure, and Interface layers
- **Go Fiber Framework** - Fast HTTP web framework
- **PostgreSQL with GORM** - Database ORM and migrations
- **Redis Cache** - High-performance caching
- **Bunny Storage Integration** - CDN file storage
- **Event-based Scheduler** - Cron job scheduling using go-co-op/gocron (not polling)
- **WebSocket Support** - Real-time communication
- **JWT Authentication** - Secure token-based auth
- **Air Hot Reload** - Development hot reload
- **Docker Support** - Containerization ready
- **Manual Dependency Injection** - Simple, clean dependency management without external frameworks

## Project Structure

```
├── cmd/api/                 # Application entry point
├── domain/                  # Business logic layer
│   ├── models/             # Data models
│   ├── repositories/       # Repository interfaces
│   └── services/           # Service interfaces
├── application/            # Application layer
│   └── serviceimpl/        # Service implementations
├── infrastructure/         # Infrastructure layer
│   ├── postgres/           # Database implementations
│   ├── redis/              # Redis client
│   ├── storage/            # File storage
│   └── websocket/          # WebSocket manager
├── interfaces/api/         # Interface layer
│   ├── handlers/           # HTTP handlers
│   ├── middleware/         # HTTP middleware
│   ├── routes/             # Route definitions (organized by domain)
│   └── websocket/          # WebSocket handlers
├── pkg/                    # Shared packages
│   ├── config/             # Configuration
│   ├── di/                 # Dependency injection
│   ├── scheduler/          # Event scheduler
│   └── utils/              # Utilities
└── .air.toml               # Air configuration
```

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL
- Redis
- Air (for hot reload): `go install github.com/cosmtrek/air@latest`

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd gofiber-template
```

2. Copy environment file:
```bash
cp .env.example .env
```

3. Update `.env` with your configuration:
```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=gofiber_template

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT Secret
JWT_SECRET=your-super-secret-jwt-key

# Bunny Storage (optional)
BUNNY_STORAGE_ZONE=your-zone
BUNNY_ACCESS_KEY=your-key
BUNNY_CDN_URL=https://your-cdn.b-cdn.net
```

4. Install dependencies:
```bash
go mod tidy
```

5. Run with hot reload:
```bash
air
```

Or run directly:
```bash
go run cmd/api/main.go
```

### Using Docker

1. Start with Docker Compose:
```bash
docker-compose up -d
```

2. The application will be available at `http://localhost:3000`

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login

### Users
- `GET /api/v1/users/profile` - Get user profile (Protected)
- `PUT /api/v1/users/profile` - Update profile (Protected)
- `DELETE /api/v1/users/profile` - Delete user (Protected)
- `GET /api/v1/users/` - List users (Admin Only)

### Tasks
- `POST /api/v1/tasks/` - Create task (Protected)
- `GET /api/v1/tasks/` - List all tasks (Admin Only)
- `GET /api/v1/tasks/my` - Get user tasks (Protected)
- `GET /api/v1/tasks/:id` - Get task by ID (Protected)
- `PUT /api/v1/tasks/:id` - Update task (Owner Only)
- `DELETE /api/v1/tasks/:id` - Delete task (Owner Only)

### Files
- `POST /api/v1/files/upload` - Upload file (Protected)
- `GET /api/v1/files/` - List all files (Admin Only)
- `GET /api/v1/files/my` - Get user files (Protected)
- `GET /api/v1/files/:id` - Get file by ID (Protected)
- `DELETE /api/v1/files/:id` - Delete file (Owner Only)

### Jobs (Scheduler)
- `POST /api/v1/jobs/` - Create scheduled job (Admin Only)
- `GET /api/v1/jobs/` - List jobs (Admin Only)
- `GET /api/v1/jobs/:id` - Get job by ID (Admin Only)
- `PUT /api/v1/jobs/:id` - Update job (Admin Only)
- `DELETE /api/v1/jobs/:id` - Delete job (Admin Only)
- `POST /api/v1/jobs/:id/start` - Start job (Admin Only)
- `POST /api/v1/jobs/:id/stop` - Stop job (Admin Only)

### WebSocket
- `GET /ws` - WebSocket connection (Optional Auth)
- Authentication via `Authorization: Bearer <token>` header
- Query parameters:
  - `room` - Room ID to join
- Supports both authenticated and anonymous connections

## WebSocket Usage

Connect to WebSocket:
```javascript
const ws = new WebSocket('ws://localhost:3000/ws?token=YOUR_JWT_TOKEN&room=ROOM_ID');

ws.onopen = function() {
    console.log('Connected to WebSocket');
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
};

// Send message
ws.send(JSON.stringify({
    type: 'ping',
    data: 'Hello Server'
}));
```

## Scheduler Usage

Create a scheduled job:
```bash
curl -X POST http://localhost:3000/api/v1/jobs \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily Backup",
    "cronExpr": "0 2 * * *",
    "payload": "{\"action\": \"backup\", \"database\": \"main\"}"
  }'
```

Cron expression examples (using gocron format):
- `0 2 * * *` - Every day at 2:00 AM
- `*/15 * * * *` - Every 15 minutes
- `0 9 * * 1-5` - Every weekday at 9:00 AM
- `@every 30s` - Every 30 seconds
- `@hourly` - Every hour
- `@daily` - Every day at midnight

## Development

### Hot Reload

Use Air for hot reload during development:
```bash
air
```

### Database Migration

Migrations run automatically on startup. Models are defined in `domain/models/`.

### Adding New Features

1. Define models in `domain/models/`
2. Create repository interface in `domain/repositories/`
3. Create service interface in `domain/services/`
4. Implement repository in `infrastructure/postgres/`
5. Implement service in `application/serviceimpl/`
6. Create handlers in `interfaces/api/handlers/`
7. Add routes in `interfaces/api/routes/` (create new route file or add to existing domain route file)
8. Register dependencies in the appropriate init method in `pkg/di/container.go`

### Manual Dependency Injection

The template uses a simple manual DI container with the following features:

- **Layered Initialization**: Each layer (config, infrastructure, repositories, services) is initialized in order
- **Graceful Shutdown**: Proper cleanup of resources on application shutdown
- **Scheduler Integration**: Event scheduler is initialized and existing jobs are loaded automatically
- **Health Checks**: Redis and database connections are tested during startup
- **Clear Logging**: Detailed startup logs show which components are initialized successfully

### Clean Architecture Handler Structure

The template follows Clean Architecture principles with organized handlers and routes:

#### Handlers Organization
- **`handlers/handlers.go`** - Central handler container with dependency injection
- **`handlers/Services`** struct - Contains all service dependencies
- **`handlers/Handlers`** struct - Contains all HTTP handlers
- **`NewHandlers()`** function - Creates handlers with proper dependency injection

#### Route Organization
Routes are organized by domain for better maintainability:

- **`routes.go`** - Main route setup function that receives `*handlers.Handlers`
- **`health_routes.go`** - Health check and root endpoints
- **`auth_routes.go`** - Authentication routes (register, login)
- **`user_routes.go`** - User management routes (profile, list users)
- **`task_routes.go`** - Task CRUD routes with authentication
- **`file_routes.go`** - File upload/download routes with authentication
- **`job_routes.go`** - Scheduler job management routes with authentication
- **`websocket_routes.go`** - WebSocket connection setup

Each route file receives the centralized `*handlers.Handlers` struct, ensuring consistent dependency injection and making the codebase more modular and easier to maintain.

### Clean Authentication & Authorization

The template implements Clean Architecture middleware patterns:

#### JWT Utilities (`pkg/utils/jwt.go`)
- **`ValidateTokenStringToUUID()`** - JWT validation and user context extraction
- **`GetUserFromContext()`** - Helper to get user from Fiber context
- **`UserContext`** struct - Clean user context with ID, username, email, role

#### Authentication Middleware (`middleware/auth.go`)
- **`Protected()`** - Validates JWT and sets user context in `c.Locals()`
- **`RequireRole(role)`** - Role-based authorization
- **`AdminOnly()`** - Admin-only access helper
- **`OwnerOnly()`** - Resource ownership validation
- **`Optional()`** - Optional authentication for public endpoints

#### Clean Handler Structure
- Handlers use `utils.GetUserFromContext(c)` instead of service dependencies
- User context is automatically available via middleware
- No service coupling in route definitions
- Consistent authentication across all protected endpoints

## Configuration

All configuration is managed through environment variables. See `.env.example` for all available options.

## Production Deployment

1. Set `APP_ENV=production`
2. Use a strong `JWT_SECRET`
3. Configure proper database credentials
4. Set up Bunny Storage for file uploads
5. Use HTTPS in production
6. Consider using a reverse proxy (nginx)

## Testing

The template includes comprehensive test examples demonstrating different testing patterns and best practices for Go Fiber applications.

### Test Structure

```
tests/examples/
├── unit/                    # Unit tests with mocks
│   ├── user_service_test.go # Service layer testing
│   └── jwt_utils_test.go    # Utility testing
├── integration/             # Integration tests with real components
│   ├── user_repository_test.go # Database integration
│   ├── auth_handler_test.go    # HTTP handler testing
│   └── websocket_test.go       # WebSocket testing
└── helpers/                 # Test utilities and helpers
    ├── test_database.go     # Database testing utilities
    ├── mock_factories.go    # Mock data factories
    └── test_server.go       # HTTP test server utilities
```

### Running Tests

Use the provided Makefile commands for different testing scenarios:

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

# Run tests with coverage report
make test-coverage

# Run specific test categories
make test-services      # Service layer tests
make test-repositories  # Repository tests
make test-handlers     # Handler tests
make test-websocket    # WebSocket tests
make test-auth         # Authentication tests
```

### Unit Testing Examples

#### Service Layer Testing (`tests/examples/unit/user_service_test.go`)

Demonstrates testing service implementations with mocked dependencies:

```go
// Test suite setup with gomock
type UserServiceTestSuite struct {
    suite.Suite
    ctrl        *gomock.Controller
    mockUserRepo *helpers.MockUserRepository
    userService  *serviceimpl.UserServiceImpl
}

// Test user registration with mock expectations
func (suite *UserServiceTestSuite) TestRegister_Success() {
    req := &models.CreateUserRequest{
        Email:     "test@example.com",
        Username:  "testuser",
        Password:  "password123",
    }

    // Mock expectations
    suite.mockUserRepo.EXPECT().
        GetByEmail(suite.ctx, req.Email).
        Return(nil, errors.New("not found")).
        Times(1)

    // Execute and assert
    user, err := suite.userService.Register(suite.ctx, req)
    assert.NoError(suite.T(), err)
    assert.Equal(suite.T(), req.Email, user.Email)
}
```

### Integration Testing Examples

#### Repository Testing (`tests/examples/integration/user_repository_test.go`)

Tests repository implementations against real database:

```go
func (suite *UserRepositoryIntegrationTestSuite) TestCreate_Success() {
    user := &models.User{
        Email:    "test@example.com",
        Username: "testuser",
        Password: "hashedpassword",
    }

    err := suite.userRepo.Create(suite.ctx, user)
    assert.NoError(suite.T(), err)

    // Verify in database
    var dbUser models.User
    result := suite.db.First(&dbUser, "id = ?", user.ID)
    assert.NoError(suite.T(), result.Error)
}
```

#### HTTP Handler Testing (`tests/examples/integration/auth_handler_test.go`)

End-to-end API testing with real HTTP requests:

```go
func (suite *AuthHandlerIntegrationTestSuite) TestRegister_Success() {
    requestBody := map[string]interface{}{
        "email":    "test@example.com",
        "username": "testuser",
        "password": "password123",
    }

    bodyBytes, _ := json.Marshal(requestBody)
    req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(bodyBytes))
    req.Header.Set("Content-Type", "application/json")

    resp, err := suite.app.Test(req)
    assert.NoError(suite.T(), err)
    assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
}
```

#### WebSocket Testing (`tests/examples/integration/websocket_test.go`)

Real-time WebSocket connection testing:

```go
func (suite *WebSocketIntegrationTestSuite) TestWebSocket_AuthenticatedConnection() {
    // Create user and JWT token
    user := createTestUser()
    token, _ := suite.userService.GenerateJWT(user)

    // Connect with authentication
    headers := http.Header{}
    headers.Add("Authorization", "Bearer "+token)

    conn, resp, err := websocket.DefaultDialer.Dial(u.String(), headers)
    assert.NoError(suite.T(), err)
    assert.Equal(suite.T(), http.StatusSwitchingProtocols, resp.StatusCode)
}
```

### Test Utilities

#### Test Database (`tests/examples/helpers/test_database.go`)

Provides utilities for database testing:

```go
// Setup in-memory SQLite for testing
testDB := helpers.NewTestDatabase()
testDB.Migrate() // Auto-migrate all models
testDB.Seed()    // Populate with test data
testDB.Clean()   // Clean all data between tests
```

#### Mock Factories (`tests/examples/helpers/mock_factories.go`)

Creates realistic test data:

```go
factory := helpers.NewMockFactory()

// Create mock user with defaults
user := factory.CreateUser()

// Create with overrides
adminUser := factory.CreateUser(func(u *models.User) {
    u.Role = "admin"
    u.Email = "admin@example.com"
})

// Create multiple users
users := factory.CreateUsers(5)
```

#### Test Server (`tests/examples/helpers/test_server.go`)

Complete HTTP test server setup:

```go
// Quick setup with authentication
server, user, token, cleanup := helpers.AuthenticatedTestServer()
defer cleanup()

// Make authenticated requests
req := httptest.NewRequest("GET", "/api/v1/users/profile", nil)
resp, err := server.MakeAuthenticatedRequest(req, token)
```

### Testing Best Practices

1. **Use Test Suites**: Leverage testify/suite for setup/teardown and shared state
2. **Mock External Dependencies**: Use gomock for service layer testing
3. **Integration Tests with Real DB**: Use SQLite in-memory for fast, isolated tests
4. **Test Utilities**: Create reusable helpers for common test scenarios
5. **Coverage Reporting**: Use `make test-coverage` to ensure adequate test coverage
6. **Parallel Testing**: Tests are designed to run concurrently with proper isolation

### Mock Generation

Generate mocks for your interfaces:

```bash
# Install mockgen
go install github.com/golang/mock/mockgen@latest

# Generate repository mocks
mockgen -source=domain/repositories/user_repository.go -destination=tests/examples/helpers/mock_user_repository.go

# Generate service mocks
mockgen -source=domain/services/user_service.go -destination=tests/examples/helpers/mock_user_service.go
```

### Continuous Integration

The Makefile includes CI-friendly commands:

```bash
make ci-test    # Run tests with race detection and coverage
make ci-lint    # Run linting with appropriate timeout
make ci-build   # Build for deployment
```

## License

This project is licensed under the MIT License.