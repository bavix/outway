package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/users"
)

const (
	DefaultAccessTokenTTLMinutes  = 30          // Increased from 5 to 30 minutes
	DefaultRefreshTokenTTLMinutes = 7 * 24 * 60 // 7 days (increased from 30 minutes)
	JWTSecretLength               = 32
	RefreshTokenLength            = 32
	CleanupIntervalMinutes        = 5
)

var (
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrUsersAlreadyExist       = errors.New("users already exist")
	ErrEmailPasswordRequired   = errors.New("email and password are required")
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrInvalidToken            = errors.New("invalid token")
)

// Service handles authentication operations.
type Service struct {
	config          *config.Config
	jwtSecret       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	refreshTokens   map[string]*RefreshToken // token -> token info
	refreshTokensMu sync.RWMutex
	authMu          sync.Mutex // protects authentication/createFirstUser operations
}

// RefreshToken represents a refresh token.
type RefreshToken struct {
	Token     string    `json:"token"`
	UserEmail string    `json:"user_email"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Claims represents JWT claims.
type Claims struct {
	jwt.RegisteredClaims

	Email string `json:"email"`
	Role  string `json:"role"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
}

// RefreshRequest represents a refresh token request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse represents a refresh token response.
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// FirstUserRequest represents a first user creation request.
type FirstUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthStatusResponse represents the authentication status response.
type AuthStatusResponse struct {
	UsersExist bool `json:"users_exist"`
}

// NewService creates a new authentication service.
func NewService(cfg *config.Config) (*Service, error) {
	var (
		secret []byte
		err    error
	)

	// Load JWT secret from config or generate new one

	if cfg.JWTSecret != "" {
		secret, err = base64.StdEncoding.DecodeString(cfg.JWTSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to decode JWT secret from config: %w", err)
		}
	} else {
		// Generate new JWT secret
		secret, err = generateJWTSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to generate JWT secret: %w", err)
		}

		// Save to config
		cfg.JWTSecret = base64.StdEncoding.EncodeToString(secret)
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("failed to save JWT secret: %w", err)
		}
	}

	// Use default TTLs (30 minutes for access, 7 days for refresh)
	accessTokenTTL := time.Duration(DefaultAccessTokenTTLMinutes) * time.Minute
	refreshTokenTTL := time.Duration(DefaultRefreshTokenTTLMinutes) * time.Minute

	service := &Service{
		config:          cfg,
		jwtSecret:       secret,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		refreshTokens:   make(map[string]*RefreshToken),
	}

	// Load refresh tokens from config
	service.loadRefreshTokensFromConfig()

	// Start cleanup goroutine for expired refresh tokens
	go service.cleanupExpiredTokens()

	return service, nil
}

// Login authenticates a user and returns a JWT token.
func (s *Service) Login(req *LoginRequest) (*LoginResponse, error) {
	s.authMu.Lock()
	defer s.authMu.Unlock()

	// Find user
	user := s.findUser(req.Email)
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	valid, err := users.VerifyPassword(req.Password, user.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}

	if !valid {
		return nil, ErrInvalidCredentials
	}

	// Generate access token
	accessToken, err := s.generateAccessToken(user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken(user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		}{
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

// CreateFirstUser creates the first user if no users exist.
func (s *Service) CreateFirstUser(req *FirstUserRequest) (*LoginResponse, error) {
	s.authMu.Lock()
	defer s.authMu.Unlock()

	// Check if users already exist
	if len(s.config.Users) > 0 {
		return nil, ErrUsersAlreadyExist
	}

	// Validate request
	if req.Email == "" || req.Password == "" {
		return nil, ErrEmailPasswordRequired
	}

	// Hash password
	hashedPassword, err := users.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &config.UserConfig{
		Email:    req.Email,
		Password: hashedPassword,
		Role:     "admin", // First user is always admin
	}

	// Add to config
	s.config.Users = append(s.config.Users, *user)

	// Save config
	if err := s.config.Save(); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	// Generate access token
	accessToken, err := s.generateAccessToken(user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken(user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		}{
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

// ValidateToken validates a JWT token and returns the claims.
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, token.Header["alg"])
		}

		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// Refresh generates a new access token using a refresh token.
func (s *Service) Refresh(req *RefreshRequest) (*RefreshResponse, error) {
	s.refreshTokensMu.RLock()
	refreshToken, exists := s.refreshTokens[req.RefreshToken]
	s.refreshTokensMu.RUnlock()

	if !exists {
		return nil, ErrInvalidToken
	}

	// Check if refresh token is expired
	if time.Now().After(refreshToken.ExpiresAt) {
		// Remove expired token
		s.refreshTokensMu.Lock()
		delete(s.refreshTokens, req.RefreshToken)
		s.refreshTokensMu.Unlock()

		return nil, ErrInvalidToken
	}

	// Find user
	user := s.findUser(refreshToken.UserEmail)
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Generate new tokens (need auth mutex for token generation)
	s.authMu.Lock()

	accessToken, err := s.generateAccessToken(user.Email, user.Role)
	if err != nil {
		s.authMu.Unlock()

		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token
	newRefreshToken, err := s.generateRefreshToken(user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Remove old refresh token
	s.refreshTokensMu.Lock()
	delete(s.refreshTokens, req.RefreshToken)
	s.refreshTokensMu.Unlock()

	// Save to config
	if err := s.saveRefreshTokensToConfig(); err != nil {
		// Log error but don't fail refresh
		_ = err
	}

	s.authMu.Unlock()

	return &RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

// generateJWTSecret generates a random JWT secret.
func generateJWTSecret() ([]byte, error) {
	secret := make([]byte, JWTSecretLength) // 256 bits

	_, err := rand.Read(secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// GetAuthStatus returns whether users exist in the system.
func (s *Service) GetAuthStatus() (*AuthStatusResponse, error) {
	return &AuthStatusResponse{
		UsersExist: len(s.config.Users) > 0,
	}, nil
}

// generateAccessToken generates an access JWT token for the given user.
func (s *Service) generateAccessToken(email, role string) (string, error) {
	claims := &Claims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "outway",
			Subject:   email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.jwtSecret)
}

// generateRefreshToken generates a refresh token for the given user.
func (s *Service) generateRefreshToken(email string) (string, error) {
	// Generate random refresh token
	tokenBytes := make([]byte, RefreshTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store refresh token
	refreshToken := &RefreshToken{
		Token:     token,
		UserEmail: email,
		ExpiresAt: time.Now().Add(s.refreshTokenTTL),
		CreatedAt: time.Now(),
	}

	s.refreshTokensMu.Lock()
	s.refreshTokens[token] = refreshToken
	s.refreshTokensMu.Unlock()

	// Save to config
	if err := s.saveRefreshTokensToConfig(); err != nil {
		// Log error but don't fail token generation
		_ = err
	}

	return token, nil
}

// loadRefreshTokensFromConfig loads refresh tokens from config.
func (s *Service) loadRefreshTokensFromConfig() {
	s.refreshTokensMu.Lock()
	defer s.refreshTokensMu.Unlock()

	now := time.Now()
	for _, rt := range s.config.RefreshTokens {
		// Only load non-expired tokens
		if now.Before(rt.ExpiresAt) {
			s.refreshTokens[rt.Token] = &RefreshToken{
				Token:     rt.Token,
				UserEmail: rt.UserEmail,
				ExpiresAt: rt.ExpiresAt,
				CreatedAt: rt.CreatedAt,
			}
		}
	}
}

// saveRefreshTokensToConfig saves refresh tokens to config.
func (s *Service) saveRefreshTokensToConfig() error {
	s.refreshTokensMu.RLock()
	defer s.refreshTokensMu.RUnlock()

	// Convert map to slice
	tokens := make([]config.RefreshToken, 0, len(s.refreshTokens))
	for _, rt := range s.refreshTokens {
		tokens = append(tokens, config.RefreshToken{
			Token:     rt.Token,
			UserEmail: rt.UserEmail,
			ExpiresAt: rt.ExpiresAt,
			CreatedAt: rt.CreatedAt,
		})
	}

	s.config.RefreshTokens = tokens

	return s.config.Save()
}

// cleanupExpiredTokens removes expired refresh tokens periodically.
func (s *Service) cleanupExpiredTokens() {
	ticker := time.NewTicker(CleanupIntervalMinutes * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		s.refreshTokensMu.Lock()

		needsSave := false

		for token, refreshToken := range s.refreshTokens {
			if now.After(refreshToken.ExpiresAt) {
				delete(s.refreshTokens, token)

				needsSave = true
			}
		}

		s.refreshTokensMu.Unlock()

		// Save to config if tokens were removed
		if needsSave {
			if err := s.saveRefreshTokensToConfig(); err != nil {
				// Log error but continue
				_ = err
			}
		}
	}
}

// findUser finds a user by email.
func (s *Service) findUser(email string) *config.UserConfig {
	for i := range s.config.Users {
		if s.config.Users[i].Email == email {
			return &s.config.Users[i]
		}
	}

	return nil
}
