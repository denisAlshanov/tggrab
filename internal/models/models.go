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
	Version          int                    `json:"version" db:"version"`
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

// Calendar Event System Models

type EventStatus string

const (
	EventStatusScheduled EventStatus = "scheduled"
	EventStatusLive      EventStatus = "live"
	EventStatusCompleted EventStatus = "completed"
	EventStatusCancelled EventStatus = "cancelled"
	EventStatusPostponed EventStatus = "postponed"
)

type Event struct {
	ID                uuid.UUID              `json:"id" db:"id"`
	ShowID            uuid.UUID              `json:"show_id" db:"show_id"`
	UserID            uuid.UUID              `json:"user_id" db:"user_id"`
	
	// Event details (can override show defaults)
	EventTitle        *string                `json:"event_title,omitempty" db:"event_title"`
	EventDescription  *string                `json:"event_description,omitempty" db:"event_description"`
	YouTubeKey        *string                `json:"youtube_key,omitempty" db:"youtube_key"`
	AdditionalKey     *string                `json:"additional_key,omitempty" db:"additional_key"`
	ZoomMeetingURL    *string                `json:"zoom_meeting_url,omitempty" db:"zoom_meeting_url"`
	ZoomMeetingID     *string                `json:"zoom_meeting_id,omitempty" db:"zoom_meeting_id"`
	ZoomPasscode      *string                `json:"zoom_passcode,omitempty" db:"zoom_passcode"`
	
	// Timing (can override show defaults)
	StartDateTime     time.Time              `json:"start_datetime" db:"start_datetime"`
	LengthMinutes     *int                   `json:"length_minutes,omitempty" db:"length_minutes"`
	EndDateTime       time.Time              `json:"end_datetime" db:"end_datetime"`
	
	// Event metadata
	Status            EventStatus            `json:"status" db:"status"`
	IsCustomized      bool                   `json:"is_customized" db:"is_customized"`
	CustomFields      map[string]interface{} `json:"custom_fields,omitempty" db:"custom_fields"`
	
	// Generation tracking
	GeneratedAt       time.Time              `json:"generated_at" db:"generated_at"`
	LastSyncedAt      *time.Time             `json:"last_synced_at,omitempty" db:"last_synced_at"`
	ShowVersion       int                    `json:"show_version" db:"show_version"`
	
	// Audit fields
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

type EventGenerationLog struct {
	ID              uuid.UUID `json:"id" db:"id"`
	ShowID          uuid.UUID `json:"show_id" db:"show_id"`
	GenerationDate  time.Time `json:"generation_date" db:"generation_date"`
	EventsGenerated int       `json:"events_generated" db:"events_generated"`
	GeneratedUntil  time.Time `json:"generated_until" db:"generated_until"`
	TriggerReason   string    `json:"trigger_reason" db:"trigger_reason"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// Event API Request/Response Types

type UpdateEventRequest struct {
	EventID          string                 `json:"event_id" binding:"required,uuid"`
	EventTitle       *string                `json:"event_title,omitempty"`
	EventDescription *string                `json:"event_description,omitempty"`
	StartDateTime    *time.Time             `json:"start_datetime,omitempty"`
	LengthMinutes    *int                   `json:"length_minutes,omitempty"`
	YouTubeKey       *string                `json:"youtube_key,omitempty"`
	AdditionalKey    *string                `json:"additional_key,omitempty"`
	ZoomMeetingURL   *string                `json:"zoom_meeting_url,omitempty"`
	ZoomMeetingID    *string                `json:"zoom_meeting_id,omitempty"`
	ZoomPasscode     *string                `json:"zoom_passcode,omitempty"`
	CustomFields     map[string]interface{} `json:"custom_fields,omitempty"`
}

type UpdateEventResponse struct {
	Success bool   `json:"success"`
	Data    *Event `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type DeleteEventRequest struct {
	EventID            string `json:"event_id" binding:"required,uuid"`
	CancellationReason string `json:"cancellation_reason,omitempty"`
}

type DeleteEventResponse struct {
	Success     bool      `json:"success"`
	Message     string    `json:"message,omitempty"`
	Data        *EventDeleteData `json:"data,omitempty"`
	Error       string    `json:"error,omitempty"`
}

type EventDeleteData struct {
	EventID     string      `json:"event_id"`
	Status      EventStatus `json:"status"`
	CancelledAt time.Time   `json:"cancelled_at"`
}

type EventListRequest struct {
	Filters    EventFilters      `json:"filters,omitempty"`
	Pagination PaginationOptions `json:"pagination,omitempty"`
	Sort       EventSortOptions  `json:"sort,omitempty"`
}

type EventFilters struct {
	Status       []EventStatus `json:"status,omitempty"`
	ShowIDs      []string      `json:"show_ids,omitempty"`
	DateRange    *DateRange    `json:"date_range,omitempty"`
	IsCustomized *bool         `json:"is_customized,omitempty"`
}

type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type EventSortOptions struct {
	Field string `json:"field,omitempty"`
	Order string `json:"order,omitempty"`
}

type EventListResponse struct {
	Success bool          `json:"success"`
	Data    *EventListData `json:"data,omitempty"`
	Error   string        `json:"error,omitempty"`
}

type EventListData struct {
	Events     []EventListItem    `json:"events"`
	Pagination PaginationResponse `json:"pagination"`
}

type EventListItem struct {
	ID              uuid.UUID   `json:"id"`
	ShowID          uuid.UUID   `json:"show_id"`
	ShowName        string      `json:"show_name"`
	EventTitle      *string     `json:"event_title,omitempty"`
	StartDateTime   time.Time   `json:"start_datetime"`
	EndDateTime     time.Time   `json:"end_datetime"`
	Status          EventStatus `json:"status"`
	IsCustomized    bool        `json:"is_customized"`
	HasZoomMeeting  bool        `json:"has_zoom_meeting"`
}

type WeekListRequest struct {
	WeekStart string       `json:"week_start" binding:"required"`
	Timezone  string       `json:"timezone,omitempty"`
	Filters   EventFilters `json:"filters,omitempty"`
}

type WeekListResponse struct {
	Success bool         `json:"success"`
	Data    *WeekListData `json:"data,omitempty"`
	Error   string       `json:"error,omitempty"`
}

type WeekListData struct {
	WeekStart    string        `json:"week_start"`
	WeekEnd      string        `json:"week_end"`
	Timezone     string        `json:"timezone"`
	Days         []WeekDay     `json:"days"`
	TotalEvents  int           `json:"total_events"`
}

type WeekDay struct {
	Date     string            `json:"date"`
	DayName  string            `json:"day_name"`
	Events   []WeekDayEvent    `json:"events"`
}

type WeekDayEvent struct {
	ID           uuid.UUID   `json:"id"`
	ShowName     string      `json:"show_name"`
	EventTitle   *string     `json:"event_title,omitempty"`
	StartTime    string      `json:"start_time"`
	EndTime      string      `json:"end_time"`
	Status       EventStatus `json:"status"`
	IsCustomized bool        `json:"is_customized"`
}

type MonthListRequest struct {
	Year     int          `json:"year" binding:"required,min=2020,max=2030"`
	Month    int          `json:"month" binding:"required,min=1,max=12"`
	Timezone string       `json:"timezone,omitempty"`
	Filters  EventFilters `json:"filters,omitempty"`
}

type MonthListResponse struct {
	Success bool           `json:"success"`
	Data    *MonthListData `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}

type MonthListData struct {
	Year           int                       `json:"year"`
	Month          int                       `json:"month"`
	MonthName      string                    `json:"month_name"`
	Timezone       string                    `json:"timezone"`
	Weeks          []MonthWeek               `json:"weeks"`
	TotalEvents    int                       `json:"total_events"`
	EventsByStatus map[EventStatus]int       `json:"events_by_status"`
}

type MonthWeek struct {
	WeekNumber int        `json:"week_number"`
	Days       []MonthDay `json:"days"`
}

type MonthDay struct {
	Date             string           `json:"date"`
	DayNumber        int              `json:"day_number"`
	IsCurrentMonth   bool             `json:"is_current_month"`
	EventsCount      int              `json:"events_count"`
	Events           []MonthDayEvent  `json:"events"`
}

type MonthDayEvent struct {
	ID              uuid.UUID   `json:"id"`
	ShowName        string      `json:"show_name"`
	StartTime       string      `json:"start_time"`
	DurationMinutes int         `json:"duration_minutes"`
	Status          EventStatus `json:"status"`
	IsCustomized    bool        `json:"is_customized"`
}

type GetEventInfoResponse struct {
	Success bool           `json:"success"`
	Data    *EventInfoData `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}

type EventInfoData struct {
	Event       *Event      `json:"event"`
	ShowDetails *ShowSummary `json:"show_details"`
}

type ShowSummary struct {
	ID            uuid.UUID     `json:"id"`
	ShowName      string        `json:"show_name"`
	RepeatPattern RepeatPattern `json:"repeat_pattern"`
	Status        ShowStatus    `json:"status"`
}

// Guest Management System Models

type ContactType string

const (
	ContactTypeEmail     ContactType = "email"
	ContactTypePhone     ContactType = "phone"
	ContactTypeTelegram  ContactType = "telegram"
	ContactTypeDiscord   ContactType = "discord"
	ContactTypeTwitter   ContactType = "twitter"
	ContactTypeLinkedIn  ContactType = "linkedin"
	ContactTypeInstagram ContactType = "instagram"
	ContactTypeWebsite   ContactType = "website"
	ContactTypeOther     ContactType = "other"
)

type GuestContact struct {
	Type      ContactType `json:"type" db:"type"`
	Value     string      `json:"value" db:"value"`
	Label     *string     `json:"label,omitempty" db:"label"`
	IsPrimary bool        `json:"is_primary" db:"is_primary"`
}

type Guest struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	UserID    uuid.UUID              `json:"user_id" db:"user_id"`
	Name      string                 `json:"name" db:"name"`
	Surname   string                 `json:"surname" db:"surname"`
	ShortName *string                `json:"short_name,omitempty" db:"short_name"`
	Contacts  []GuestContact         `json:"contacts,omitempty" db:"contacts"`
	Notes     *string                `json:"notes,omitempty" db:"notes"`
	Avatar    *string                `json:"avatar,omitempty" db:"avatar"`
	Tags      []string               `json:"tags,omitempty" db:"tags"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
}

// Guest API Request/Response Types

type CreateGuestRequest struct {
	Name      string                 `json:"name" binding:"required,min=1,max=255"`
	Surname   string                 `json:"surname" binding:"required,min=1,max=255"`
	ShortName *string                `json:"short_name,omitempty"`
	Contacts  []GuestContact         `json:"contacts,omitempty"`
	Notes     *string                `json:"notes,omitempty"`
	Avatar    *string                `json:"avatar,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type CreateGuestResponse struct {
	Success bool   `json:"success"`
	Data    *Guest `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type UpdateGuestRequest struct {
	GuestID   string                 `json:"guest_id" binding:"required,uuid"`
	Name      *string                `json:"name,omitempty"`
	Surname   *string                `json:"surname,omitempty"`
	ShortName *string                `json:"short_name,omitempty"`
	Contacts  []GuestContact         `json:"contacts,omitempty"`
	Notes     *string                `json:"notes,omitempty"`
	Avatar    *string                `json:"avatar,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateGuestResponse struct {
	Success bool   `json:"success"`
	Data    *Guest `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type DeleteGuestRequest struct {
	GuestID string `json:"guest_id" binding:"required,uuid"`
}

type DeleteGuestResponse struct {
	Success bool             `json:"success"`
	Message string           `json:"message,omitempty"`
	Data    *GuestDeleteData `json:"data,omitempty"`
	Error   string           `json:"error,omitempty"`
}

type GuestDeleteData struct {
	GuestID   string    `json:"guest_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

type ListGuestsRequest struct {
	Filters    GuestFilters      `json:"filters,omitempty"`
	Pagination PaginationOptions `json:"pagination,omitempty"`
	Sort       GuestSortOptions  `json:"sort,omitempty"`
}

type GuestFilters struct {
	Search           string        `json:"search,omitempty"`
	Tags             []string      `json:"tags,omitempty"`
	HasContactType   []ContactType `json:"has_contact_type,omitempty"`
	CreatedDateRange *DateRange    `json:"created_date_range,omitempty"`
}

type GuestSortOptions struct {
	Field string `json:"field,omitempty"`
	Order string `json:"order,omitempty"`
}

type ListGuestsResponse struct {
	Success bool           `json:"success"`
	Data    *ListGuestsData `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}

type ListGuestsData struct {
	Guests     []GuestListItem    `json:"guests"`
	Pagination PaginationResponse `json:"pagination"`
}

type GuestListItem struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Surname      string    `json:"surname"`
	ShortName    *string   `json:"short_name,omitempty"`
	PrimaryEmail *string   `json:"primary_email,omitempty"`
	Avatar       *string   `json:"avatar,omitempty"`
	Tags         []string  `json:"tags"`
	NotesPreview *string   `json:"notes_preview,omitempty"`
	ContactCount int       `json:"contact_count"`
	CreatedAt    time.Time `json:"created_at"`
}

type AutocompleteResponse struct {
	Success bool              `json:"success"`
	Data    *AutocompleteData `json:"data,omitempty"`
	Error   string            `json:"error,omitempty"`
}

type AutocompleteData struct {
	Suggestions  []GuestSuggestion `json:"suggestions"`
	Query        string            `json:"query"`
	TotalMatches int               `json:"total_matches"`
}

type GuestSuggestion struct {
	ID             uuid.UUID     `json:"id"`
	DisplayName    string        `json:"display_name"`
	Name           string        `json:"name"`
	Surname        string        `json:"surname"`
	ShortName      *string       `json:"short_name,omitempty"`
	Avatar         *string       `json:"avatar,omitempty"`
	PrimaryContact *GuestContact `json:"primary_contact,omitempty"`
	Tags           []string      `json:"tags"`
	MatchScore     float64       `json:"match_score"`
}

type GetGuestInfoResponse struct {
	Success bool           `json:"success"`
	Data    *GuestInfoData `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}

type GuestInfoData struct {
	Guest *Guest      `json:"guest"`
	Stats *GuestStats `json:"stats,omitempty"`
}

type GuestStats struct {
	TotalShows      int        `json:"total_shows"`
	LastAppearance  *time.Time `json:"last_appearance,omitempty"`
	UpcomingShows   int        `json:"upcoming_shows"`
}
