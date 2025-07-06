package models

import (
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	ContentID           string     `json:"content_id" db:"content_id"`
	TelegramLink        string     `json:"telegram_link" db:"telegram_link"`
	ChannelName         string     `json:"channel_name" db:"channel_name"`
	OriginalChannelName string     `json:"original_channel_name" db:"original_channel_name"`
	MessageID           int64      `json:"message_id" db:"message_id"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
	Status              PostStatus `json:"status" db:"status"`
	MediaCount          int        `json:"media_count" db:"media_count"`
	TotalSize           int64      `json:"total_size" db:"total_size"`
	ErrorMessage        *string    `json:"error_message,omitempty" db:"error_message"`
}

type PostStatus string

const (
	PostStatusPending    PostStatus = "pending"
	PostStatusProcessing PostStatus = "processing"
	PostStatusCompleted  PostStatus = "completed"
	PostStatusFailed     PostStatus = "failed"
)

type Media struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	MediaID          string                 `json:"media_id" db:"media_id"`
	ContentID        string                 `json:"content_id" db:"content_id"`
	TelegramFileID   string                 `json:"telegram_file_id" db:"telegram_file_id"`
	FileName         string                 `json:"file_name" db:"file_name"`
	OriginalFileName string                 `json:"original_file_name" db:"original_file_name"`
	FileType         string                 `json:"file_type" db:"file_type"`
	FileSize         int64                  `json:"file_size" db:"file_size"`
	S3Bucket         string                 `json:"s3_bucket" db:"s3_bucket"`
	S3Key            string                 `json:"s3_key" db:"s3_key"`
	FileHash         string                 `json:"file_hash" db:"file_hash"`
	DownloadedAt     time.Time              `json:"downloaded_at" db:"downloaded_at"`
	Metadata         map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

type PaginationOptions struct {
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Sort  string `json:"sort"`
}

type PostListResponse struct {
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
	Links []PostListItem `json:"links"`
}

type PostListItem struct {
	ContentID  string     `json:"content_id"`
	Link       string     `json:"link"`
	AddedAt    time.Time  `json:"added_at"`
	MediaCount int        `json:"media_count"`
	Status     PostStatus `json:"status"`
}

type MediaListResponse struct {
	ContentID  string          `json:"content_id"`
	Link       string          `json:"link"`
	MediaFiles []MediaListItem `json:"media_files"`
}

type MediaListItem struct {
	MediaID    string    `json:"media_id"`
	FileName   string    `json:"file_name"`
	FileType   string    `json:"file_type"`
	FileSize   int64     `json:"file_size"`
	UploadDate time.Time `json:"upload_date"`
}

type AddPostRequest struct {
	Link string `json:"link" binding:"required"`
}

type AddPostResponse struct {
	Status           string     `json:"status"`
	Message          string     `json:"message"`
	ContentID        string     `json:"content_id"`
	MediaCount       int        `json:"media_count"`
	ProcessingStatus PostStatus `json:"processing_status"`
}

type GetLinkListRequest struct {
	ContentID string `json:"content_id" binding:"required"`
}

type GetLinkMediaRequest struct {
	MediaID string `json:"media_id" binding:"required"`
}

type GetLinkMediaURIRequest struct {
	MediaID       string `json:"media_id" binding:"required"`
	ExpiryMinutes int    `json:"expiry_minutes,omitempty"`
}

type GetLinkMediaURIResponse struct {
	MediaID   string    `json:"media_id"`
	S3URL     string    `json:"s3_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type UpdateMediaRequest struct {
	MediaID          string                 `json:"media_id" binding:"required"`
	FileName         *string                `json:"file_name,omitempty"`
	OriginalFileName *string                `json:"original_file_name,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateMediaResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	MediaID string `json:"media_id"`
}

type DeleteMediaRequest struct {
	MediaID string `json:"media_id" binding:"required"`
}

type DeleteMediaResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	MediaID string `json:"media_id"`
}

// Show types
type RepeatPattern string

const (
	RepeatNone      RepeatPattern = "none"
	RepeatDaily     RepeatPattern = "daily"
	RepeatWeekly    RepeatPattern = "weekly"
	RepeatBiweekly  RepeatPattern = "biweekly"
	RepeatMonthly   RepeatPattern = "monthly"
	RepeatCustom    RepeatPattern = "custom"
)

type ShowStatus string

const (
	ShowStatusActive    ShowStatus = "active"
	ShowStatusPaused    ShowStatus = "paused"
	ShowStatusCompleted ShowStatus = "completed"
	ShowStatusCancelled ShowStatus = "cancelled"
)

// Scheduling configuration types
type MonthlyWeekNumber int

const (
	MonthlyWeekFirst  MonthlyWeekNumber = 1
	MonthlyWeekSecond MonthlyWeekNumber = 2
	MonthlyWeekThird  MonthlyWeekNumber = 3
	MonthlyWeekFourth MonthlyWeekNumber = 4
	MonthlyWeekLast   MonthlyWeekNumber = -1
)

type MonthlyDayFallback string

const (
	MonthlyDayFallbackLastDay MonthlyDayFallback = "last_day"
	MonthlyDayFallbackSkip    MonthlyDayFallback = "skip"
)

type SchedulingConfig struct {
	// For weekly and biweekly patterns
	Weekdays []int `json:"weekdays,omitempty"`
	
	// For monthly patterns - weekday-based
	MonthlyWeekday    *int `json:"monthly_weekday,omitempty"`    // 0=Sunday, 1=Monday, etc.
	MonthlyWeekNumber *int `json:"monthly_week_number,omitempty"` // 1, 2, 3, 4, or -1 for last
	
	// For monthly patterns - calendar day-based
	MonthlyDay         *int    `json:"monthly_day,omitempty"`         // 1-31
	MonthlyDayFallback *string `json:"monthly_day_fallback,omitempty"` // "last_day", "skip"
}

type Show struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	ShowName         string                 `json:"show_name" db:"show_name"`
	YouTubeKey       string                 `json:"youtube_key" db:"youtube_key"`
	AdditionalKey    *string                `json:"additional_key,omitempty" db:"additional_key"`
	ZoomMeetingURL   *string                `json:"zoom_meeting_url,omitempty" db:"zoom_meeting_url"`
	ZoomMeetingID    *string                `json:"zoom_meeting_id,omitempty" db:"zoom_meeting_id"`
	ZoomPasscode     *string                `json:"zoom_passcode,omitempty" db:"zoom_passcode"`
	StartTime        time.Time              `json:"start_time" db:"start_time"`
	LengthMinutes    int                    `json:"length_minutes" db:"length_minutes"`
	FirstEventDate   time.Time              `json:"first_event_date" db:"first_event_date"`
	RepeatPattern    RepeatPattern          `json:"repeat_pattern" db:"repeat_pattern"`
	SchedulingConfig *SchedulingConfig      `json:"scheduling_config,omitempty" db:"scheduling_config"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	Status           ShowStatus             `json:"status" db:"status"`
	UserID           uuid.UUID              `json:"user_id" db:"user_id"`
	Metadata         map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// Show request/response types
type CreateShowRequest struct {
	ShowName         string                 `json:"show_name" binding:"required,min=1,max=255"`
	YouTubeKey       string                 `json:"youtube_key" binding:"required"`
	AdditionalKey    *string                `json:"additional_key,omitempty"`
	ZoomMeetingURL   *string                `json:"zoom_meeting_url,omitempty"`
	ZoomMeetingID    *string                `json:"zoom_meeting_id,omitempty"`
	ZoomPasscode     *string                `json:"zoom_passcode,omitempty"`
	StartTime        string                 `json:"start_time" binding:"required"`
	LengthMinutes    int                    `json:"length_minutes" binding:"required,min=1,max=1440"`
	FirstEventDate   string                 `json:"first_event_date" binding:"required"`
	RepeatPattern    RepeatPattern          `json:"repeat_pattern" binding:"required"`
	SchedulingConfig *SchedulingConfig      `json:"scheduling_config,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

type CreateShowResponse struct {
	Success bool   `json:"success"`
	Data    *Show  `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type DeleteShowRequest struct {
	ShowID string `json:"show_id" binding:"required,uuid"`
}

type DeleteShowResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ListShowsRequest struct {
	Filters    ListShowsFilters    `json:"filters,omitempty"`
	Pagination PaginationOptions   `json:"pagination,omitempty"`
	Sort       ListShowsSortOptions `json:"sort,omitempty"`
}

type ListShowsFilters struct {
	Status        []ShowStatus    `json:"status,omitempty"`
	RepeatPattern []RepeatPattern `json:"repeat_pattern,omitempty"`
	Search        string          `json:"search,omitempty"`
}

type ListShowsSortOptions struct {
	Field string `json:"field,omitempty"`
	Order string `json:"order,omitempty"`
}

type ShowListItem struct {
	ID               uuid.UUID         `json:"id"`
	ShowName         string            `json:"show_name"`
	StartTime        time.Time         `json:"start_time"`
	LengthMinutes    int               `json:"length_minutes"`
	FirstEventDate   time.Time         `json:"first_event_date"`
	RepeatPattern    RepeatPattern     `json:"repeat_pattern"`
	SchedulingConfig *SchedulingConfig `json:"scheduling_config,omitempty"`
	Status           ShowStatus        `json:"status"`
	HasZoomMeeting   bool              `json:"has_zoom_meeting"`
	NextOccurrence   *time.Time        `json:"next_occurrence,omitempty"`
	NextOccurrences  []time.Time       `json:"next_occurrences,omitempty"`
}

type ListShowsResponse struct {
	Success    bool                  `json:"success"`
	Data       *ListShowsData        `json:"data,omitempty"`
	Error      string                `json:"error,omitempty"`
}

type ListShowsData struct {
	Shows      []ShowListItem     `json:"shows"`
	Pagination PaginationResponse `json:"pagination"`
}

type PaginationResponse struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type GetShowInfoResponse struct {
	Success bool              `json:"success"`
	Data    *ShowInfoData     `json:"data,omitempty"`
	Error   string            `json:"error,omitempty"`
}

type ShowInfoData struct {
	Show           *Show         `json:"show"`
	UpcomingEvents []ShowEvent   `json:"upcoming_events"`
}

type ShowEvent struct {
	Date          string    `json:"date"`
	StartDateTime time.Time `json:"start_datetime"`
	EndDateTime   time.Time `json:"end_datetime"`
}
