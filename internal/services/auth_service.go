package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"archive-system/internal/config"
	"archive-system/internal/models"
	"archive-system/internal/repository"
)

// ─── Errors ───────────────────────────────────────────────────
var (
	ErrInvalidCredentials = errors.New("username or password is incorrect")
	ErrUserNotActive      = errors.New("user account is not active")
	ErrUsernameExists     = errors.New("username already exists")
	ErrEmailExists        = errors.New("email already exists")
	ErrOrgNotFound        = errors.New("organization not found")
)

// ─── JWT Claims ───────────────────────────────────────────────
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	OrgID    string `json:"org_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// ─── Request / Response structs ───────────────────────────────
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	OrgCode  string `json:"org_code"  binding:"required"`
	Username string `json:"username"  binding:"required,min=3,max=50"`
	Email    string `json:"email"     binding:"required,email"`
	Password string `json:"password"  binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
}

type AuthResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      *models.User `json:"user"`
}

// ─── Service ──────────────────────────────────────────────────
type AuthService struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{userRepo: userRepo, cfg: cfg}
}

// Login - تسجيل دخول
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	// جيب اليوزر من الـ DB
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	if !user.IsActive {
		return nil, ErrUserNotActive
	}

	// تأكد من الـ password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// حدّث last_login_at
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	// اعمل JWT token
	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// امسح الـ password hash قبل ما ترجعه
	user.PasswordHash = ""

	return &AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

// Register - تسجيل يوزر جديد
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// تأكد إن الـ organization موجودة
	org, err := s.userRepo.GetOrganizationByCode(ctx, req.OrgCode)
	if err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}
	if org == nil {
		return nil, ErrOrgNotFound
	}

	// تأكد إن الـ username مش موجود
	existing, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameExists
	}

	// تأكد إن الـ email مش موجود
	existingEmail, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existingEmail != nil {
		return nil, ErrEmailExists
	}

	// Hash الـ password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// عمل اليوزر
	user := &models.User{
		OrganizationID: org.ID,
		Username:       req.Username,
		Email:          req.Email,
		PasswordHash:   string(hash),
		FullName:       req.FullName,
		Role:           "member",
		IsActive:       true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// اعمل JWT token
	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	user.PasswordHash = ""

	return &AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

// generateToken - يعمل JWT token
func (s *AuthService) generateToken(user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(time.Duration(s.cfg.JWT.ExpiryHours) * time.Hour)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		OrgID:    user.OrganizationID,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

// ValidateToken - تحقق من الـ token
func (s *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
