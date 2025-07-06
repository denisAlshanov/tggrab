package utils

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength is the minimum required password length
	MinPasswordLength = 8
	// MaxPasswordLength is the maximum allowed password length (bcrypt limit)
	MaxPasswordLength = 72
	// DefaultBcryptCost is the default bcrypt cost factor
	DefaultBcryptCost = 10
)

var (
	// ErrPasswordTooShort is returned when password is too short
	ErrPasswordTooShort = errors.New("password must be at least 8 characters long")
	// ErrPasswordTooLong is returned when password exceeds bcrypt limit
	ErrPasswordTooLong = errors.New("password must not exceed 72 characters")
	// ErrPasswordTooWeak is returned when password doesn't meet complexity requirements
	ErrPasswordTooWeak = errors.New("password must contain at least one uppercase letter, one lowercase letter, one digit, and one special character")
	// ErrInvalidPassword is returned when password verification fails
	ErrInvalidPassword = errors.New("invalid password")
)

// PasswordComplexity defines password complexity requirements
type PasswordComplexity struct {
	RequireUppercase bool
	RequireLowercase bool
	RequireDigit     bool
	RequireSpecial   bool
}

// DefaultPasswordComplexity returns the default password complexity requirements
func DefaultPasswordComplexity() PasswordComplexity {
	return PasswordComplexity{
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSpecial:   true,
	}
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	// Validate password length
	if len(password) < MinPasswordLength {
		return "", ErrPasswordTooShort
	}
	if len(password) > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}

	// Generate hash
	hash, err := bcrypt.GenerateFromPassword([]byte(password), DefaultBcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidPassword
		}
		return fmt.Errorf("failed to verify password: %w", err)
	}
	return nil
}

// ValidatePasswordComplexity validates password against complexity requirements
func ValidatePasswordComplexity(password string, complexity PasswordComplexity) error {
	// Check length
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)

	// Check each character
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	// Build error message for missing requirements
	var missing []string
	if complexity.RequireUppercase && !hasUpper {
		missing = append(missing, "uppercase letter")
	}
	if complexity.RequireLowercase && !hasLower {
		missing = append(missing, "lowercase letter")
	}
	if complexity.RequireDigit && !hasDigit {
		missing = append(missing, "digit")
	}
	if complexity.RequireSpecial && !hasSpecial {
		missing = append(missing, "special character")
	}

	if len(missing) > 0 {
		return fmt.Errorf("password must contain at least one %s", strings.Join(missing, ", "))
	}

	return nil
}

// ValidateAndHashPassword validates password complexity and returns hash
func ValidateAndHashPassword(password string) (string, error) {
	// Validate complexity with default requirements
	if err := ValidatePasswordComplexity(password, DefaultPasswordComplexity()); err != nil {
		return "", err
	}

	// Hash the password
	return HashPassword(password)
}

// NeedsRehash checks if a password hash needs to be updated
func NeedsRehash(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return true
	}
	return cost < DefaultBcryptCost
}

// GenerateSecurePassword generates a cryptographically secure random password
func GenerateSecurePassword(length int) (string, error) {
	if length < MinPasswordLength {
		length = MinPasswordLength
	}
	if length > MaxPasswordLength {
		length = MaxPasswordLength
	}

	// Character sets
	const (
		uppercaseLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowercaseLetters = "abcdefghijklmnopqrstuvwxyz"
		digits           = "0123456789"
		specialChars     = "!@#$%^&*()_+-=[]{}|;:,.<>?"
	)

	// Ensure at least one character from each required set
	password := make([]byte, 0, length)
	password = append(password, uppercaseLetters[RandomInt(len(uppercaseLetters))])
	password = append(password, lowercaseLetters[RandomInt(len(lowercaseLetters))])
	password = append(password, digits[RandomInt(len(digits))])
	password = append(password, specialChars[RandomInt(len(specialChars))])

	// Fill the rest with random characters from all sets
	allChars := uppercaseLetters + lowercaseLetters + digits + specialChars
	for i := 4; i < length; i++ {
		password = append(password, allChars[RandomInt(len(allChars))])
	}

	// Shuffle the password
	for i := len(password) - 1; i > 0; i-- {
		j := RandomInt(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}