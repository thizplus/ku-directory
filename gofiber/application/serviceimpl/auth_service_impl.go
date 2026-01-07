package serviceimpl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"gofiber-template/infrastructure/oauth"
)

type AuthServiceImpl struct {
	userRepo    repositories.UserRepository
	googleOAuth *oauth.GoogleOAuth
	jwtSecret   string
}

func NewAuthService(
	userRepo repositories.UserRepository,
	googleOAuth *oauth.GoogleOAuth,
	jwtSecret string,
) services.AuthService {
	return &AuthServiceImpl{
		userRepo:    userRepo,
		googleOAuth: googleOAuth,
		jwtSecret:   jwtSecret,
	}
}

func (s *AuthServiceImpl) GetGoogleAuthURL(state string) string {
	return s.googleOAuth.GetAuthURL(state)
}

func (s *AuthServiceImpl) HandleGoogleCallback(ctx context.Context, code string) (string, *models.User, error) {
	// Exchange code for tokens
	tokenResp, err := s.googleOAuth.ExchangeCode(ctx, code)
	if err != nil {
		return "", nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from Google
	userInfo, err := s.googleOAuth.GetUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Find or create user
	user, err := s.findOrCreateGoogleUser(ctx, userInfo)
	if err != nil {
		return "", nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	// Update last login and save Drive tokens
	now := time.Now()
	user.LastLogin = &now

	// Debug: Log token info
	fmt.Printf("🔑 TokenResp AccessToken length: %d\n", len(tokenResp.AccessToken))
	fmt.Printf("🔑 TokenResp RefreshToken length: %d\n", len(tokenResp.RefreshToken))
	fmt.Printf("🔑 TokenResp ExpiresIn: %d\n", tokenResp.ExpiresIn)

	user.DriveAccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		user.DriveRefreshToken = tokenResp.RefreshToken
	}
	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	user.DriveTokenExpiry = &expiry
	user.UpdatedAt = now

	if err := s.userRepo.Update(ctx, user.ID, user); err != nil {
		fmt.Printf("❌ Failed to update user tokens: %v\n", err)
	} else {
		fmt.Printf("✅ User tokens updated successfully\n")
	}

	// Generate JWT token
	token, err := s.generateJWT(user)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, user, nil
}

func (s *AuthServiceImpl) GetCurrentUser(ctx context.Context, tokenString string) (*models.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userIDStr, ok := claims["user_id"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid token claims")
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID in token")
		}

		return s.userRepo.GetByID(ctx, userID)
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *AuthServiceImpl) findOrCreateGoogleUser(ctx context.Context, info *oauth.GoogleUserInfo) (*models.User, error) {
	// Try to find existing user by provider ID
	user, err := s.userRepo.GetByProviderID(ctx, "google", info.ID)
	if err == nil {
		// User exists, update info if needed
		updated := false
		if user.Avatar != info.Picture && info.Picture != "" {
			user.Avatar = info.Picture
			updated = true
		}
		if user.FirstName != info.GivenName && info.GivenName != "" {
			user.FirstName = info.GivenName
			updated = true
		}
		if user.LastName != info.FamilyName && info.FamilyName != "" {
			user.LastName = info.FamilyName
			updated = true
		}
		if updated {
			user.UpdatedAt = time.Now()
			s.userRepo.Update(ctx, user.ID, user)
		}
		return user, nil
	}

	// Try to find existing user by email
	user, err = s.userRepo.GetByEmail(ctx, info.Email)
	if err == nil {
		// Link Google account to existing user
		user.Provider = "google"
		user.ProviderID = info.ID
		if user.Avatar == "" && info.Picture != "" {
			user.Avatar = info.Picture
		}
		user.UpdatedAt = time.Now()
		if err := s.userRepo.Update(ctx, user.ID, user); err != nil {
			return nil, err
		}
		return user, nil
	}

	// Create new user
	username := s.generateUsername(info.Email, info.GivenName)
	now := time.Now()

	// Generate unique public slug
	publicSlug := s.generatePublicSlug(info.GivenName, info.FamilyName)

	newUser := &models.User{
		ID:         uuid.New(),
		Email:      info.Email,
		Username:   username,
		FirstName:  info.GivenName,
		LastName:   info.FamilyName,
		Avatar:     info.Picture,
		Provider:   "google",
		ProviderID: info.ID,
		Role:       "user",
		IsActive:   true,
		LastLogin:  &now,
		PublicSlug: publicSlug,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *AuthServiceImpl) generateUsername(email, givenName string) string {
	// Generate username from email or name
	base := strings.Split(email, "@")[0]
	if givenName != "" {
		base = strings.ToLower(givenName)
	}
	// Remove special characters
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, strings.ToLower(base))

	if len(base) < 3 {
		base = "user"
	}

	// Add random suffix to ensure uniqueness
	return fmt.Sprintf("%s_%s", base, uuid.New().String()[:8])
}

func (s *AuthServiceImpl) generatePublicSlug(givenName, familyName string) string {
	// Generate public slug from name
	base := strings.ToLower(givenName)
	if familyName != "" {
		base = base + "-" + strings.ToLower(familyName)
	}
	// Remove special characters, keep only letters, numbers, and hyphens
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, base)

	if len(base) < 3 {
		base = "user"
	}

	// Add random suffix to ensure uniqueness
	return fmt.Sprintf("%s-%s", base, uuid.New().String()[:8])
}

func (s *AuthServiceImpl) generateJWT(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID.String(),
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
		"provider": user.Provider,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
