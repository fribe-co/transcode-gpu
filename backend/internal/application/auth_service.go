package application

import (
	"errors"
	"time"

	"github.com/cashbacktv/backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo        domain.UserRepository
	jwtSecret       []byte
	tokenExpiration time.Duration
	refreshExpiration time.Duration
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Claims represents JWT claims
type Claims struct {
	UserID uuid.UUID       `json:"user_id"`
	Email  string          `json:"email"`
	Role   domain.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo domain.UserRepository, jwtSecret string, tokenExpHours, refreshExpHours int) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		jwtSecret:       []byte(jwtSecret),
		tokenExpiration: time.Duration(tokenExpHours) * time.Hour,
		refreshExpiration: time.Duration(refreshExpHours) * time.Hour,
	}
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(email, password string) (*TokenPair, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.generateTokenPair(user)
}

// RefreshToken validates refresh token and returns new token pair
func (s *AuthService) RefreshToken(refreshToken string) (*TokenPair, error) {
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return s.generateTokenPair(user)
}

// ValidateToken validates a JWT token
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetCurrentUser retrieves user from token
func (s *AuthService) GetCurrentUser(tokenString string) (*domain.User, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	return s.userRepo.GetByID(claims.UserID)
}

// CreateUser creates a new user with hashed password
func (s *AuthService) CreateUser(email, password, name string, role domain.UserRole) (*domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := domain.NewUser(email, name, role)
	user.PasswordHash = string(hash)

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// generateTokenPair creates access and refresh tokens
func (s *AuthService) generateTokenPair(user *domain.User) (*TokenPair, error) {
	now := time.Now()
	expiresAt := now.Add(s.tokenExpiration)

	accessClaims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "cashbacktv",
			Subject:   user.ID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	refreshClaims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "cashbacktv",
			Subject:   user.ID.String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    expiresAt,
	}, nil
}





