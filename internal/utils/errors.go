package utils

import (
	"fmt"
	"net/http"
)

type ErrorCode string

const (
	ErrorCodeInvalidLinkFormat ErrorCode = "INVALID_LINK_FORMAT"
	ErrorCodePostNotFound      ErrorCode = "POST_NOT_FOUND"
	ErrorCodeMediaNotFound     ErrorCode = "MEDIA_NOT_FOUND"
	ErrorCodeDownloadFailed    ErrorCode = "DOWNLOAD_FAILED"
	ErrorCodeS3UploadFailed    ErrorCode = "S3_UPLOAD_FAILED"
	ErrorCodeDatabaseError     ErrorCode = "DATABASE_ERROR"
	ErrorCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrorCodeUnauthorized      ErrorCode = "UNAUTHORIZED"
	ErrorCodeInternalError     ErrorCode = "INTERNAL_ERROR"
	ErrorCodeValidationError   ErrorCode = "VALIDATION_ERROR"
	ErrorCodeDuplicatePost     ErrorCode = "DUPLICATE_POST"
)

type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	StatusCode int                    `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func NewError(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Details:    make(map[string]interface{}),
	}
}

func NewErrorWithDetails(code ErrorCode, message string, statusCode int, details map[string]interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Details:    details,
	}
}

// Common error constructors
func NewValidationError(message string, details map[string]interface{}) *AppError {
	return NewErrorWithDetails(ErrorCodeValidationError, message, http.StatusBadRequest, details)
}

func NewInvalidLinkError(link string) *AppError {
	return NewErrorWithDetails(
		ErrorCodeInvalidLinkFormat,
		"The provided Telegram link is not in a valid format",
		http.StatusBadRequest,
		map[string]interface{}{
			"expected_format": "https://t.me/channel_name/post_id",
			"provided":        link,
		},
	)
}

func NewPostNotFoundError(postID string) *AppError {
	return NewError(
		ErrorCodePostNotFound,
		fmt.Sprintf("Post with ID %s not found", postID),
		http.StatusNotFound,
	)
}

func NewMediaNotFoundError(mediaID string) *AppError {
	return NewError(
		ErrorCodeMediaNotFound,
		fmt.Sprintf("Media with ID %s not found", mediaID),
		http.StatusNotFound,
	)
}

func NewDatabaseError(err error) *AppError {
	return NewError(
		ErrorCodeDatabaseError,
		"Database operation failed",
		http.StatusInternalServerError,
	)
}

func NewS3Error(err error) *AppError {
	return NewError(
		ErrorCodeS3UploadFailed,
		"Failed to upload to S3",
		http.StatusInternalServerError,
	)
}

func NewDownloadError(err error) *AppError {
	return NewError(
		ErrorCodeDownloadFailed,
		"Failed to download media from Telegram",
		http.StatusInternalServerError,
	)
}

func NewUnauthorizedError() *AppError {
	return NewError(
		ErrorCodeUnauthorized,
		"Invalid or missing authentication",
		http.StatusUnauthorized,
	)
}

func NewRateLimitError() *AppError {
	return NewError(
		ErrorCodeRateLimitExceeded,
		"Too many requests",
		http.StatusTooManyRequests,
	)
}

func NewInternalError() *AppError {
	return NewError(
		ErrorCodeInternalError,
		"An unexpected error occurred",
		http.StatusInternalServerError,
	)
}

func NewDuplicatePostError(link string) *AppError {
	return NewErrorWithDetails(
		ErrorCodeDuplicatePost,
		"Post already exists",
		http.StatusConflict,
		map[string]interface{}{
			"link": link,
		},
	)
}
