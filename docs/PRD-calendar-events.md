# Product Requirements Document: Calendar Event System

## 1. Executive Summary

### 1.1 Purpose
This PRD defines the requirements for implementing a calendar event system that generates concrete, manageable events from show templates, enabling fine-grained control over individual show instances while maintaining synchronization with parent shows.

### 1.2 Scope
The calendar event system will:
- Generate concrete events from show scheduling patterns
- Allow individual event customization without affecting the parent show
- Provide automated event generation and maintenance
- Support event cancellation while preserving show templates
- Offer comprehensive event management APIs
- Enable calendar views (daily, weekly, monthly)

### 1.3 Business Value
- **Granular Control**: Manage individual show instances separately from templates
- **Calendar Integration**: Provide calendar-like views for better schedule management
- **Flexibility**: Override event properties without affecting future shows
- **Automation**: Automatic event generation reduces manual overhead
- **User Experience**: Familiar calendar interface for content creators

## 2. Problem Statement

### 2.1 Current Limitations
- Shows are templates without concrete scheduled instances
- Cannot modify individual show occurrences without affecting all future shows
- No calendar view of upcoming scheduled content
- Manual tracking of specific show dates and customizations

### 2.2 Desired Solution
- Concrete calendar events generated from show templates
- Individual event modification capabilities
- Automated event generation and maintenance
- Calendar-style APIs for different time views
- Event-specific customizations (titles, descriptions, special guests, etc.)

## 3. User Stories

### 3.1 Event Management
1. **As a content creator**, I want events to be automatically created from my show templates so that I can see my concrete schedule.

2. **As a content creator**, I want to customize individual events (change title, add special guests, modify duration) without affecting my show template.

3. **As a content creator**, I want to cancel specific events while keeping my show template active for future dates.

4. **As a content creator**, I want to see my events in calendar views (daily, weekly, monthly) for better schedule planning.

### 3.2 Automation
5. **As a content creator**, I want events to be automatically generated for the next 3 months so that I always have a populated calendar.

6. **As a content creator**, I want events to be updated when I modify my show template, with the option to preserve individual customizations.

7. **As a system**, I want to automatically maintain the event pipeline so that the calendar is always current.

## 4. Functional Requirements

### 4.1 Data Model

#### 4.1.1 Event Entity
```go
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
    EndDateTime       time.Time              `json:"end_datetime" db:"end_datetime"` // Calculated field
    
    // Event metadata
    Status            EventStatus            `json:"status" db:"status"`
    IsCustomized      bool                   `json:"is_customized" db:"is_customized"`
    CustomFields      map[string]interface{} `json:"custom_fields,omitempty" db:"custom_fields"`
    
    // Generation tracking
    GeneratedAt       time.Time              `json:"generated_at" db:"generated_at"`
    LastSyncedAt      *time.Time             `json:"last_synced_at,omitempty" db:"last_synced_at"`
    ShowVersion       int                    `json:"show_version" db:"show_version"` // Track show changes
    
    // Audit fields
    CreatedAt         time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

type EventStatus string

const (
    EventStatusScheduled EventStatus = "scheduled"
    EventStatusLive      EventStatus = "live"
    EventStatusCompleted EventStatus = "completed"
    EventStatusCancelled EventStatus = "cancelled"
    EventStatusPostponed EventStatus = "postponed"
)
```

#### 4.1.2 Event Generation Metadata
```go
type EventGenerationLog struct {
    ID              uuid.UUID `json:"id" db:"id"`
    ShowID          uuid.UUID `json:"show_id" db:"show_id"`
    GenerationDate  time.Time `json:"generation_date" db:"generation_date"`
    EventsGenerated int       `json:"events_generated" db:"events_generated"`
    GeneratedUntil  time.Time `json:"generated_until" db:"generated_until"`
    TriggerReason   string    `json:"trigger_reason" db:"trigger_reason"` // "new_show", "show_update", "maintenance"
    CreatedAt       time.Time `json:"created_at" db:"created_at"`
}
```

### 4.2 Database Schema

#### 4.2.1 Events Table
```sql
-- Create event status enum
CREATE TYPE event_status AS ENUM ('scheduled', 'live', 'completed', 'cancelled', 'postponed');

-- Create events table
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    show_id UUID NOT NULL,
    user_id UUID NOT NULL,
    
    -- Event details (nullable = uses show defaults)
    event_title VARCHAR(500),
    event_description TEXT,
    youtube_key VARCHAR(500),
    additional_key VARCHAR(500),
    zoom_meeting_url VARCHAR(500),
    zoom_meeting_id VARCHAR(100),
    zoom_passcode VARCHAR(50),
    
    -- Timing
    start_datetime TIMESTAMP WITH TIME ZONE NOT NULL,
    length_minutes INTEGER,
    end_datetime TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Event metadata
    status event_status DEFAULT 'scheduled',
    is_customized BOOLEAN DEFAULT FALSE,
    custom_fields JSONB,
    
    -- Generation tracking
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_synced_at TIMESTAMP WITH TIME ZONE,
    show_version INTEGER DEFAULT 1,
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign keys
    FOREIGN KEY (show_id) REFERENCES shows(id) ON DELETE CASCADE,
    
    -- Constraints
    CONSTRAINT valid_event_timing CHECK (end_datetime > start_datetime),
    CONSTRAINT valid_length CHECK (length_minutes IS NULL OR length_minutes > 0)
);

-- Create indexes
CREATE INDEX idx_events_show_id ON events(show_id);
CREATE INDEX idx_events_user_id ON events(user_id);
CREATE INDEX idx_events_start_datetime ON events(start_datetime);
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_events_date_range ON events(start_datetime, end_datetime);
CREATE INDEX idx_events_user_status ON events(user_id, status);

-- Create update trigger
CREATE TRIGGER update_events_updated_at 
BEFORE UPDATE ON events
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();
```

#### 4.2.2 Event Generation Log Table
```sql
CREATE TABLE event_generation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    show_id UUID NOT NULL,
    generation_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    events_generated INTEGER NOT NULL,
    generated_until TIMESTAMP WITH TIME ZONE NOT NULL,
    trigger_reason VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (show_id) REFERENCES shows(id) ON DELETE CASCADE
);

CREATE INDEX idx_generation_logs_show_id ON event_generation_logs(show_id);
CREATE INDEX idx_generation_logs_date ON event_generation_logs(generation_date);
```

### 4.3 Event Generation Logic

#### 4.3.1 Generation Triggers
Events are generated automatically in these scenarios:

1. **New Show Creation**: Generate events for next 3 full months
2. **Show Update**: Regenerate future events (preserve customizations)
3. **Maintenance Job**: Every 10 minutes, ensure 3-month horizon
4. **Manual Trigger**: Admin endpoint for manual generation

#### 4.3.2 Generation Algorithm
```go
func GenerateEventsForShow(show *models.Show, generateUntil time.Time) ([]models.Event, error) {
    var events []models.Event
    
    // Calculate next 3 full months from today
    now := time.Now()
    endOfCurrentMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)
    threeMonthsLater := endOfCurrentMonth.AddDate(0, 3, 0)
    
    // Get show occurrences using advanced scheduling
    occurrences := utils.CalculateNextOccurrences(show, 1000) // Large number to cover 3 months
    
    for _, occurrence := range occurrences {
        if occurrence.After(threeMonthsLater) {
            break // Stop after 3 months
        }
        
        event := models.Event{
            ShowID:        show.ID,
            UserID:        show.UserID,
            StartDateTime: occurrence,
            EndDateTime:   occurrence.Add(time.Duration(show.LengthMinutes) * time.Minute),
            Status:        models.EventStatusScheduled,
            IsCustomized:  false,
            ShowVersion:   show.Version, // Track show version for sync
            GeneratedAt:   now,
        }
        
        events = append(events, event)
    }
    
    return events, nil
}
```

#### 4.3.3 Event Synchronization
When shows are updated:
1. **Preserve Customizations**: Keep user-modified events as-is
2. **Update Non-Customized**: Sync with new show properties
3. **Add New Events**: Generate new events if scheduling changed
4. **Remove Obsolete**: Mark events as cancelled if no longer valid

### 4.4 API Endpoints

#### 4.4.1 Update Event
**Endpoint**: `PUT /api/v1/event/update`

**Request Body**:
```json
{
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "event_title": "Special Episode: Year End Review",
    "event_description": "Join us for a special year-end review episode",
    "start_datetime": "2025-01-15T14:00:00Z",
    "length_minutes": 90,
    "youtube_key": "special-stream-key",
    "zoom_meeting_url": "https://zoom.us/j/special-meeting",
    "custom_fields": {
        "special_guests": ["John Doe", "Jane Smith"],
        "episode_theme": "year_end_review",
        "sponsor": "TechCorp"
    }
}
```

**Response**:
```json
{
    "success": true,
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "show_id": "660e8400-e29b-41d4-a716-446655440001",
        "event_title": "Special Episode: Year End Review",
        "start_datetime": "2025-01-15T14:00:00Z",
        "end_datetime": "2025-01-15T15:30:00Z",
        "status": "scheduled",
        "is_customized": true,
        "updated_at": "2025-01-07T10:00:00Z"
    }
}
```

#### 4.4.2 Delete Event
**Endpoint**: `DELETE /api/v1/event/delete`

**Request Body**:
```json
{
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "cancellation_reason": "Conflicting schedule"
}
```

**Response**:
```json
{
    "success": true,
    "message": "Event cancelled successfully",
    "data": {
        "event_id": "550e8400-e29b-41d4-a716-446655440000",
        "status": "cancelled",
        "cancelled_at": "2025-01-07T10:00:00Z"
    }
}
```

#### 4.4.3 List Events
**Endpoint**: `POST /api/v1/event/list`

**Request Body**:
```json
{
    "filters": {
        "status": ["scheduled", "live"],
        "show_ids": ["660e8400-e29b-41d4-a716-446655440001"],
        "date_range": {
            "start": "2025-01-01T00:00:00Z",
            "end": "2025-01-31T23:59:59Z"
        },
        "is_customized": true
    },
    "pagination": {
        "page": 1,
        "limit": 50
    },
    "sort": {
        "field": "start_datetime",
        "order": "asc"
    }
}
```

**Response**:
```json
{
    "success": true,
    "data": {
        "events": [
            {
                "id": "550e8400-e29b-41d4-a716-446655440000",
                "show_id": "660e8400-e29b-41d4-a716-446655440001",
                "show_name": "Weekly Tech Talk",
                "event_title": "Special Episode: Year End Review",
                "start_datetime": "2025-01-15T14:00:00Z",
                "end_datetime": "2025-01-15T15:30:00Z",
                "status": "scheduled",
                "is_customized": true,
                "has_zoom_meeting": true
            }
        ],
        "pagination": {
            "page": 1,
            "limit": 50,
            "total": 12,
            "total_pages": 1
        }
    }
}
```

#### 4.4.4 Week List Events
**Endpoint**: `POST /api/v1/event/weekList`

**Request Body**:
```json
{
    "week_start": "2025-01-13", // Monday of the week
    "timezone": "America/New_York",
    "filters": {
        "status": ["scheduled", "live"],
        "show_ids": ["660e8400-e29b-41d4-a716-446655440001"]
    }
}
```

**Response**:
```json
{
    "success": true,
    "data": {
        "week_start": "2025-01-13",
        "week_end": "2025-01-19",
        "timezone": "America/New_York",
        "days": [
            {
                "date": "2025-01-13",
                "day_name": "Monday",
                "events": []
            },
            {
                "date": "2025-01-15",
                "day_name": "Wednesday", 
                "events": [
                    {
                        "id": "550e8400-e29b-41d4-a716-446655440000",
                        "show_name": "Weekly Tech Talk",
                        "event_title": "Special Episode: Year End Review",
                        "start_time": "14:00",
                        "end_time": "15:30",
                        "status": "scheduled",
                        "is_customized": true
                    }
                ]
            }
        ],
        "total_events": 1
    }
}
```

#### 4.4.5 Month List Events
**Endpoint**: `POST /api/v1/event/monthList`

**Request Body**:
```json
{
    "year": 2025,
    "month": 1,
    "timezone": "America/New_York",
    "filters": {
        "status": ["scheduled", "live", "completed"]
    }
}
```

**Response**:
```json
{
    "success": true,
    "data": {
        "year": 2025,
        "month": 1,
        "month_name": "January",
        "timezone": "America/New_York",
        "weeks": [
            {
                "week_number": 1,
                "days": [
                    {
                        "date": "2025-01-01",
                        "day_number": 1,
                        "is_current_month": true,
                        "events_count": 0,
                        "events": []
                    },
                    {
                        "date": "2025-01-15",
                        "day_number": 15,
                        "is_current_month": true,
                        "events_count": 1,
                        "events": [
                            {
                                "id": "550e8400-e29b-41d4-a716-446655440000",
                                "show_name": "Weekly Tech Talk",
                                "start_time": "14:00",
                                "duration_minutes": 90,
                                "status": "scheduled",
                                "is_customized": true
                            }
                        ]
                    }
                ]
            }
        ],
        "total_events": 12,
        "events_by_status": {
            "scheduled": 10,
            "completed": 2,
            "cancelled": 0
        }
    }
}
```

#### 4.4.6 Get Event Info
**Endpoint**: `GET /api/v1/event/info/{event_id}`

**Response**:
```json
{
    "success": true,
    "data": {
        "event": {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "show_id": "660e8400-e29b-41d4-a716-446655440001",
            "show_name": "Weekly Tech Talk",
            "event_title": "Special Episode: Year End Review",
            "event_description": "Join us for a special year-end review episode",
            "start_datetime": "2025-01-15T14:00:00Z",
            "end_datetime": "2025-01-15T15:30:00Z",
            "length_minutes": 90,
            "status": "scheduled",
            "is_customized": true,
            "youtube_key": "special-stream-key",
            "zoom_meeting_url": "https://zoom.us/j/special-meeting",
            "custom_fields": {
                "special_guests": ["John Doe", "Jane Smith"],
                "episode_theme": "year_end_review",
                "sponsor": "TechCorp"
            },
            "generated_at": "2025-01-01T10:00:00Z",
            "last_synced_at": "2025-01-05T10:00:00Z",
            "created_at": "2025-01-01T10:00:00Z",
            "updated_at": "2025-01-07T10:00:00Z"
        },
        "show_details": {
            "id": "660e8400-e29b-41d4-a716-446655440001",
            "show_name": "Weekly Tech Talk",
            "repeat_pattern": "weekly",
            "status": "active"
        }
    }
}
```

## 5. Event Generation & Maintenance

### 5.1 Automated Generation Scenarios

#### 5.1.1 New Show Creation
```go
func OnShowCreated(show *models.Show) error {
    events, err := GenerateEventsForShow(show, getThreeMonthHorizon())
    if err != nil {
        return err
    }
    
    err = db.CreateEvents(ctx, events)
    if err != nil {
        return err
    }
    
    // Log generation
    log := models.EventGenerationLog{
        ShowID:          show.ID,
        EventsGenerated: len(events),
        GeneratedUntil:  getThreeMonthHorizon(),
        TriggerReason:   "new_show",
    }
    return db.CreateGenerationLog(ctx, log)
}
```

#### 5.1.2 Show Update
```go
func OnShowUpdated(oldShow, newShow *models.Show) error {
    // Get existing future events
    futureEvents, err := db.GetFutureEvents(ctx, newShow.ID)
    if err != nil {
        return err
    }
    
    // Separate customized from non-customized events
    customizedEvents := filterCustomizedEvents(futureEvents)
    nonCustomizedEvents := filterNonCustomizedEvents(futureEvents)
    
    // Cancel old non-customized events
    err = db.CancelEvents(ctx, nonCustomizedEvents)
    if err != nil {
        return err
    }
    
    // Generate new events
    newEvents, err := GenerateEventsForShow(newShow, getThreeMonthHorizon())
    if err != nil {
        return err
    }
    
    // Merge with existing customized events
    finalEvents := mergeEvents(customizedEvents, newEvents)
    
    return db.ReplaceNonCustomizedEvents(ctx, newShow.ID, finalEvents)
}
```

#### 5.1.3 Maintenance Job (Every 10 Minutes)
```go
func MaintenanceEventGeneration() error {
    activeShows, err := db.GetActiveShows(ctx)
    if err != nil {
        return err
    }
    
    threeMonthHorizon := getThreeMonthHorizon()
    
    for _, show := range activeShows {
        lastGenerated, err := db.GetLastGenerationDate(ctx, show.ID)
        if err != nil {
            continue
        }
        
        // Check if we need to generate more events
        if lastGenerated.Before(threeMonthHorizon) {
            newEvents, err := GenerateEventsForShow(&show, threeMonthHorizon)
            if err != nil {
                logError("Failed to generate events for show", show.ID, err)
                continue
            }
            
            // Only insert events that don't already exist
            uniqueEvents := filterExistingEvents(newEvents, show.ID)
            if len(uniqueEvents) > 0 {
                err = db.CreateEvents(ctx, uniqueEvents)
                if err != nil {
                    logError("Failed to create events", show.ID, err)
                    continue
                }
                
                // Log generation
                log := models.EventGenerationLog{
                    ShowID:          show.ID,
                    EventsGenerated: len(uniqueEvents),
                    GeneratedUntil:  threeMonthHorizon,
                    TriggerReason:   "maintenance",
                }
                db.CreateGenerationLog(ctx, log)
            }
        }
    }
    
    return nil
}
```

### 5.2 Show Cancellation Handling
```go
func OnShowCancelled(show *models.Show) error {
    // Cancel all future events for this show
    futureEvents, err := db.GetFutureEvents(ctx, show.ID)
    if err != nil {
        return err
    }
    
    now := time.Now()
    var eventsToCancel []uuid.UUID
    
    for _, event := range futureEvents {
        if event.StartDateTime.After(now) && event.Status == models.EventStatusScheduled {
            eventsToCancel = append(eventsToCancel, event.ID)
        }
    }
    
    return db.CancelEventsByIDs(ctx, eventsToCancel, "show_cancelled")
}
```

## 6. Data Relationships & Inheritance

### 6.1 Event-Show Relationship
Events inherit default values from their parent show but can override any field:

| Field | Inheritance Rule |
|-------|------------------|
| Title | Uses show name if `event_title` is null |
| Description | Uses show metadata if `event_description` is null |
| YouTube Key | Uses show YouTube key if `youtube_key` is null |
| Zoom Settings | Uses show Zoom settings if event Zoom fields are null |
| Duration | Uses show length if `length_minutes` is null |
| Timing | Calculated from show scheduling, can be overridden |

### 6.2 Customization Tracking
- `is_customized` flag tracks if event has user modifications
- `show_version` tracks which version of show the event was generated from
- `last_synced_at` tracks when event was last synchronized with show

### 6.3 Version Management
```go
func GetEffectiveEventData(event *models.Event, show *models.Show) EventData {
    return EventData{
        Title:          coalesce(event.EventTitle, &show.ShowName),
        Description:    coalesce(event.EventDescription, getShowDescription(show)),
        YouTubeKey:     coalesce(event.YouTubeKey, &show.YouTubeKey),
        ZoomMeetingURL: coalesce(event.ZoomMeetingURL, show.ZoomMeetingURL),
        Duration:       coalesceInt(event.LengthMinutes, &show.LengthMinutes),
        StartTime:      event.StartDateTime, // Always from event
        EndTime:        event.EndDateTime,   // Always calculated
    }
}
```

## 7. Performance Considerations

### 7.1 Database Optimization
- **Partitioning**: Partition events table by date for better query performance
- **Indexing**: Composite indexes for common query patterns
- **Archiving**: Archive completed events older than 1 year

### 7.2 Generation Efficiency
- **Batch Processing**: Generate events in batches to avoid memory issues
- **Incremental Updates**: Only generate missing events, not full regeneration
- **Background Jobs**: Use queue system for event generation

### 7.3 Caching Strategy
```go
type EventCache struct {
    UserDayEvents    map[string][]Event // Cache day views
    UserWeekEvents   map[string][]Event // Cache week views
    UserMonthEvents  map[string][]Event // Cache month views
    TTL              time.Duration
}
```

## 8. Error Handling & Edge Cases

### 8.1 Generation Conflicts
- **Duplicate Events**: Detect and prevent duplicate event creation
- **Timezone Issues**: Handle DST transitions and timezone changes
- **Show Deletion**: Gracefully handle show deletion with existing events

### 8.2 Data Consistency
- **Orphaned Events**: Clean up events for deleted shows
- **Sync Failures**: Retry mechanisms for failed synchronizations
- **Concurrent Updates**: Handle concurrent event modifications

### 8.3 Validation Rules
```go
func ValidateEvent(event *models.Event, show *models.Show) error {
    // Event must be in the future (for new events)
    if event.StartDateTime.Before(time.Now()) && event.Status == EventStatusScheduled {
        return errors.New("cannot schedule events in the past")
    }
    
    // Event must belong to user
    if event.UserID != show.UserID {
        return errors.New("event user must match show user")
    }
    
    // Duration must be reasonable
    duration := event.EndDateTime.Sub(event.StartDateTime)
    if duration <= 0 || duration > 24*time.Hour {
        return errors.New("event duration must be between 1 minute and 24 hours")
    }
    
    return nil
}
```

## 9. Migration Strategy

### 9.1 Database Migration
```sql
-- Migration Version 7: Add events and generation tracking tables
CREATE TYPE event_status AS ENUM ('scheduled', 'live', 'completed', 'cancelled', 'postponed');

-- Events table creation
CREATE TABLE events (
    -- Full table definition as specified above
);

-- Generation logs table
CREATE TABLE event_generation_logs (
    -- Full table definition as specified above
);

-- Add version tracking to shows table
ALTER TABLE shows ADD COLUMN version INTEGER DEFAULT 1;
```

### 9.2 Initial Data Population
```go
func MigrateExistingShows() error {
    activeShows, err := db.GetAllActiveShows(ctx)
    if err != nil {
        return err
    }
    
    for _, show := range activeShows {
        // Generate events for each existing show
        events, err := GenerateEventsForShow(&show, getThreeMonthHorizon())
        if err != nil {
            log.Printf("Failed to generate events for show %s: %v", show.ID, err)
            continue
        }
        
        err = db.CreateEvents(ctx, events)
        if err != nil {
            log.Printf("Failed to create events for show %s: %v", show.ID, err)
            continue
        }
        
        log.Printf("Generated %d events for show %s", len(events), show.ShowName)
    }
    
    return nil
}
```

## 10. API Response Models

### 10.1 Request/Response Types
```go
type UpdateEventRequest struct {
    EventID          string                 `json:"event_id" binding:"required,uuid"`
    EventTitle       *string                `json:"event_title,omitempty"`
    EventDescription *string                `json:"event_description,omitempty"`
    StartDateTime    *time.Time             `json:"start_datetime,omitempty"`
    LengthMinutes    *int                   `json:"length_minutes,omitempty"`
    YouTubeKey       *string                `json:"youtube_key,omitempty"`
    ZoomMeetingURL   *string                `json:"zoom_meeting_url,omitempty"`
    ZoomMeetingID    *string                `json:"zoom_meeting_id,omitempty"`
    ZoomPasscode     *string                `json:"zoom_passcode,omitempty"`
    CustomFields     map[string]interface{} `json:"custom_fields,omitempty"`
}

type EventListRequest struct {
    Filters    EventFilters          `json:"filters,omitempty"`
    Pagination PaginationOptions     `json:"pagination,omitempty"`
    Sort       EventSortOptions      `json:"sort,omitempty"`
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

type WeekListRequest struct {
    WeekStart string       `json:"week_start" binding:"required"` // YYYY-MM-DD format
    Timezone  string       `json:"timezone,omitempty"`
    Filters   EventFilters `json:"filters,omitempty"`
}

type MonthListRequest struct {
    Year     int          `json:"year" binding:"required,min=2020,max=2030"`
    Month    int          `json:"month" binding:"required,min=1,max=12"`
    Timezone string       `json:"timezone,omitempty"`
    Filters  EventFilters `json:"filters,omitempty"`
}
```

## 11. Testing Requirements

### 11.1 Unit Tests
- Event generation algorithms
- Date calculation logic
- Inheritance and override logic
- Validation functions

### 11.2 Integration Tests
- End-to-end event lifecycle
- Show-event synchronization
- API endpoint functionality
- Database operations

### 11.3 Performance Tests
- Event generation at scale
- Calendar view query performance
- Concurrent event modifications
- Maintenance job efficiency

### 11.4 Edge Case Tests
- DST transitions
- Leap years
- Timezone changes
- Show schedule modifications
- Bulk event operations

## 12. Monitoring & Observability

### 12.1 Metrics
- Events generated per hour/day
- Event generation success/failure rates
- API response times for calendar views
- Database query performance
- Cache hit rates

### 12.2 Alerts
- Event generation failures
- High API latency for calendar views
- Database connection issues
- Maintenance job failures

### 12.3 Logging
```go
type EventOperationLog struct {
    Operation     string    `json:"operation"`      // "create", "update", "cancel", "generate"
    EventID       string    `json:"event_id,omitempty"`
    ShowID        string    `json:"show_id,omitempty"`
    UserID        string    `json:"user_id"`
    Success       bool      `json:"success"`
    ErrorMessage  string    `json:"error_message,omitempty"`
    Duration      int64     `json:"duration_ms"`
    Timestamp     time.Time `json:"timestamp"`
}
```

## 13. Future Enhancements

### 13.1 Phase 2 Features
- **Recurring Event Modifications**: Apply changes to all future instances
- **Event Templates**: Create reusable event templates
- **Conflict Detection**: Detect and warn about scheduling conflicts
- **Automatic Rescheduling**: AI-powered rescheduling suggestions

### 13.2 Phase 3 Features
- **External Calendar Integration**: Sync with Google Calendar, Outlook
- **Collaborative Events**: Multi-user event management
- **Event Analytics**: Viewership and engagement tracking
- **Mobile Calendar Views**: Native mobile calendar interface

### 13.3 Advanced Features
- **Smart Scheduling**: ML-powered optimal time slot suggestions
- **Audience Timezone Optimization**: Schedule based on audience location
- **Dynamic Event Content**: AI-generated event descriptions and titles
- **Cross-Platform Publishing**: Automatic event creation on multiple platforms

## 14. Success Metrics

### 14.1 Technical Metrics
- **Event Generation Accuracy**: 99.9% successful generation rate
- **API Performance**: <100ms for calendar views, <200ms for event operations
- **Data Consistency**: 100% parent-child data integrity
- **System Reliability**: 99.95% uptime for event services

### 14.2 User Experience Metrics
- **Calendar View Usage**: Monthly active users using calendar views
- **Event Customization Rate**: Percentage of events customized by users
- **User Satisfaction**: Calendar functionality satisfaction scores
- **Feature Adoption**: Time to first calendar view after registration

## 15. Implementation Timeline

### 15.1 Phase 1 (Week 1-3): Core Foundation
- Database schema design and migration
- Basic event data models
- Event generation algorithms
- Core CRUD operations

### 15.2 Phase 2 (Week 4-6): API Development
- Event management endpoints
- Calendar view APIs
- Event-show synchronization
- Validation and error handling

### 15.3 Phase 3 (Week 7-8): Automation & Polish
- Maintenance job implementation
- Performance optimization
- Testing and documentation
- Deployment and monitoring

## 16. Risk Assessment

### 16.1 Technical Risks
- **Performance**: Large-scale event generation might impact performance
  - *Mitigation*: Implement efficient algorithms and background processing
- **Data Consistency**: Complex parent-child relationships
  - *Mitigation*: Comprehensive validation and transaction management
- **Storage Growth**: Events table will grow rapidly
  - *Mitigation*: Implement archiving and partitioning strategies

### 16.2 User Experience Risks
- **Complexity**: Users might find event vs show distinction confusing
  - *Mitigation*: Clear UI/UX design and comprehensive documentation
- **Performance**: Calendar views might be slow with many events
  - *Mitigation*: Implement caching and pagination strategies