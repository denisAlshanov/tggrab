# Product Requirements Document: Show Planning Feature

## 1. Executive Summary

### 1.1 Purpose
This PRD defines the requirements for adding show planning functionality to the stPlaner service, enabling users to create, manage, and schedule recurring YouTube live shows with integrated Zoom meeting support.

### 1.2 Scope
The feature will allow users to:
- Create planned shows with scheduling information
- Set up recurring show patterns
- Manage show metadata including YouTube integration keys
- Perform CRUD operations via RESTful API

## 2. Feature Overview

### 2.1 Business Value
- **Automation**: Streamline the process of planning recurring YouTube shows
- **Organization**: Centralize show planning and scheduling
- **Scalability**: Support multiple shows with different recurring patterns
- **Integration**: Direct connection to YouTube for automated show creation and Zoom for virtual meetings

### 2.2 User Stories
1. As a content creator, I want to create a recurring show schedule so that I can automate my streaming workflow
2. As a content creator, I want to view all my planned shows so that I can manage my content calendar
3. As a content creator, I want to delete shows that are no longer needed
4. As a content creator, I want to view detailed information about a specific show
5. As a content creator, I want to have a default Zoom meeting link for each show so attendees can join virtually

## 3. Functional Requirements

### 3.1 Data Model

#### Show Entity
```go
type Show struct {
    ID              uuid.UUID           `json:"id" db:"id"`
    ShowName        string              `json:"show_name" db:"show_name"`
    YouTubeKey      string              `json:"youtube_key" db:"youtube_key"`
    AdditionalKey   string              `json:"additional_key" db:"additional_key"`
    ZoomMeetingURL  string              `json:"zoom_meeting_url" db:"zoom_meeting_url"`
    ZoomMeetingID   string              `json:"zoom_meeting_id" db:"zoom_meeting_id"`
    ZoomPasscode    string              `json:"zoom_passcode" db:"zoom_passcode"`
    StartTime       time.Time           `json:"start_time" db:"start_time"`
    LengthMinutes   int                 `json:"length_minutes" db:"length_minutes"`
    FirstEventDate  time.Time           `json:"first_event_date" db:"first_event_date"`
    RepeatPattern   RepeatPattern       `json:"repeat_pattern" db:"repeat_pattern"`
    CreatedAt       time.Time           `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time           `json:"updated_at" db:"updated_at"`
    Status          ShowStatus          `json:"status" db:"status"`
    UserID          uuid.UUID           `json:"user_id" db:"user_id"`
    Metadata        json.RawMessage     `json:"metadata" db:"metadata"`
}
```

#### Supporting Types
```go
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
```

### 3.2 Database Schema

```sql
-- Create custom types
CREATE TYPE repeat_pattern AS ENUM ('none', 'daily', 'weekly', 'biweekly', 'monthly', 'custom');
CREATE TYPE show_status AS ENUM ('active', 'paused', 'completed', 'cancelled');

-- Create shows table
CREATE TABLE shows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    show_name VARCHAR(255) NOT NULL,
    youtube_key VARCHAR(500) NOT NULL,
    additional_key VARCHAR(500),
    zoom_meeting_url VARCHAR(500),
    zoom_meeting_id VARCHAR(100),
    zoom_passcode VARCHAR(50),
    start_time TIME NOT NULL,
    length_minutes INTEGER NOT NULL CHECK (length_minutes > 0),
    first_event_date DATE NOT NULL,
    repeat_pattern repeat_pattern NOT NULL DEFAULT 'none',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status show_status DEFAULT 'active',
    user_id UUID NOT NULL,
    metadata JSONB,
    
    -- Constraints
    CONSTRAINT valid_first_event_date CHECK (first_event_date >= CURRENT_DATE),
    CONSTRAINT valid_length CHECK (length_minutes BETWEEN 1 AND 1440), -- Max 24 hours
    CONSTRAINT valid_zoom_url CHECK (zoom_meeting_url IS NULL OR zoom_meeting_url ~ '^https://.*\.zoom\.us/.*')
);

-- Create indexes
CREATE INDEX idx_shows_user_id ON shows(user_id);
CREATE INDEX idx_shows_status ON shows(status);
CREATE INDEX idx_shows_first_event_date ON shows(first_event_date);
CREATE INDEX idx_shows_show_name ON shows(LOWER(show_name));
CREATE INDEX idx_shows_zoom_meeting_id ON shows(zoom_meeting_id) WHERE zoom_meeting_id IS NOT NULL;

-- Create update trigger
CREATE TRIGGER update_shows_updated_at 
BEFORE UPDATE ON shows
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();
```

### 3.3 API Endpoints

#### 3.3.1 Create Show
**Endpoint**: `POST /api/v1/show/create`

**Request Body**:
```json
{
    "show_name": "Weekly Tech Talk",
    "youtube_key": "youtube-stream-key-here",
    "additional_key": "backup-stream-key",
    "zoom_meeting_url": "https://company.zoom.us/j/1234567890?pwd=abc123",
    "zoom_meeting_id": "123 456 7890",
    "zoom_passcode": "abc123",
    "start_time": "14:00:00",
    "length_minutes": 60,
    "first_event_date": "2025-01-15",
    "repeat_pattern": "weekly",
    "metadata": {
        "description": "Weekly technology discussions",
        "tags": ["tech", "programming", "news"]
    }
}
```

**Response**:
```json
{
    "success": true,
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "show_name": "Weekly Tech Talk",
        "youtube_key": "youtube-stream-key-here",
        "additional_key": "backup-stream-key",
        "zoom_meeting_url": "https://company.zoom.us/j/1234567890?pwd=abc123",
        "zoom_meeting_id": "123 456 7890",
        "zoom_passcode": "abc123",
        "start_time": "14:00:00",
        "length_minutes": 60,
        "first_event_date": "2025-01-15",
        "repeat_pattern": "weekly",
        "status": "active",
        "created_at": "2025-01-07T10:00:00Z",
        "updated_at": "2025-01-07T10:00:00Z"
    }
}
```

**Validation Rules**:
- `show_name`: Required, 1-255 characters
- `youtube_key`: Required, valid YouTube stream key format
- `start_time`: Required, valid time format (HH:MM:SS)
- `length_minutes`: Required, 1-1440 (24 hours max)
- `first_event_date`: Required, must be today or future date
- `repeat_pattern`: Required, must be valid enum value
- `zoom_meeting_url`: Optional, must be valid Zoom URL format (https://*.zoom.us/*)
- `zoom_meeting_id`: Optional, alphanumeric with optional spaces/dashes
- `zoom_passcode`: Optional, max 50 characters

#### 3.3.2 Delete Show
**Endpoint**: `DELETE /api/v1/show/delete`

**Request Body**:
```json
{
    "show_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response**:
```json
{
    "success": true,
    "message": "Show deleted successfully"
}
```

**Business Logic**:
- Soft delete by setting status to 'cancelled'
- Preserve show history for analytics
- Cancel any pending scheduled events

#### 3.3.3 List Shows
**Endpoint**: `POST /api/v1/show/list`

**Request Body**:
```json
{
    "filters": {
        "status": ["active", "paused"],
        "repeat_pattern": ["weekly", "monthly"],
        "search": "tech"
    },
    "pagination": {
        "page": 1,
        "limit": 20
    },
    "sort": {
        "field": "first_event_date",
        "order": "asc"
    }
}
```

**Response**:
```json
{
    "success": true,
    "data": {
        "shows": [
            {
                "id": "550e8400-e29b-41d4-a716-446655440000",
                "show_name": "Weekly Tech Talk",
                "start_time": "14:00:00",
                "length_minutes": 60,
                "first_event_date": "2025-01-15",
                "repeat_pattern": "weekly",
                "status": "active",
                "has_zoom_meeting": true,
                "next_occurrence": "2025-01-22T14:00:00Z"
            }
        ],
        "pagination": {
            "page": 1,
            "limit": 20,
            "total": 45,
            "total_pages": 3
        }
    }
}
```

#### 3.3.4 Get Show Info
**Endpoint**: `GET /api/v1/show/info/{show_id}`

**Response**:
```json
{
    "success": true,
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "show_name": "Weekly Tech Talk",
        "youtube_key": "youtube-stream-key-here",
        "additional_key": "backup-stream-key",
        "zoom_meeting_url": "https://company.zoom.us/j/1234567890?pwd=abc123",
        "zoom_meeting_id": "123 456 7890",
        "zoom_passcode": "abc123",
        "start_time": "14:00:00",
        "length_minutes": 60,
        "first_event_date": "2025-01-15",
        "repeat_pattern": "weekly",
        "status": "active",
        "created_at": "2025-01-07T10:00:00Z",
        "updated_at": "2025-01-07T10:00:00Z",
        "metadata": {
            "description": "Weekly technology discussions",
            "tags": ["tech", "programming", "news"],
            "zoom_settings": {
                "waiting_room": true,
                "auto_recording": "cloud"
            }
        },
        "upcoming_events": [
            {
                "date": "2025-01-22",
                "start_datetime": "2025-01-22T14:00:00Z",
                "end_datetime": "2025-01-22T15:00:00Z"
            },
            {
                "date": "2025-01-29",
                "start_datetime": "2025-01-29T14:00:00Z",
                "end_datetime": "2025-01-29T15:00:00Z"
            }
        ]
    }
}
```

## 4. Non-Functional Requirements

### 4.1 Performance
- API response time < 200ms for single show operations
- List endpoint should support pagination for large datasets
- Database queries should use appropriate indexes

### 4.2 Security
- JWT authentication required for all endpoints
- YouTube keys and Zoom credentials should be encrypted at rest
- Zoom passcodes should be hashed before storage
- Rate limiting: 100 requests per minute per user
- Zoom URLs should be validated to prevent phishing

### 4.3 Scalability
- Support for up to 1000 shows per user
- Efficient query patterns for recurring event calculations
- Caching strategy for frequently accessed shows

### 4.4 Data Validation
- Input validation on all endpoints
- Timezone handling for international users
- Prevent scheduling conflicts

## 5. Technical Implementation

### 5.1 Dependencies
```go
// New dependencies to add
go get github.com/robfig/cron/v3  // For cron pattern parsing
```

### 5.2 Migration
```sql
-- Migration Version 5: Add shows table
CREATE TYPE repeat_pattern AS ENUM ('none', 'daily', 'weekly', 'biweekly', 'monthly', 'custom');
CREATE TYPE show_status AS ENUM ('active', 'paused', 'completed', 'cancelled');

CREATE TABLE shows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    show_name VARCHAR(255) NOT NULL,
    youtube_key VARCHAR(500) NOT NULL,
    additional_key VARCHAR(500),
    zoom_meeting_url VARCHAR(500),
    zoom_meeting_id VARCHAR(100),
    zoom_passcode VARCHAR(50),
    start_time TIME NOT NULL,
    length_minutes INTEGER NOT NULL CHECK (length_minutes > 0),
    first_event_date DATE NOT NULL,
    repeat_pattern repeat_pattern NOT NULL DEFAULT 'none',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status show_status DEFAULT 'active',
    user_id UUID NOT NULL,
    metadata JSONB,
    
    -- Constraints
    CONSTRAINT valid_first_event_date CHECK (first_event_date >= CURRENT_DATE),
    CONSTRAINT valid_length CHECK (length_minutes BETWEEN 1 AND 1440),
    CONSTRAINT valid_zoom_url CHECK (zoom_meeting_url IS NULL OR zoom_meeting_url ~ '^https://.*\.zoom\.us/.*')
);
```

### 5.3 Handler Structure
```
internal/api/handlers/
└── shows.go          // All show-related handlers
    ├── CreateShow
    ├── DeleteShow
    ├── ListShows
    └── GetShowInfo
```

## 6. Future Considerations

### 6.1 Phase 2 Features
- Integration with YouTube API for automatic stream creation
- Email/SMS notifications for upcoming shows
- Show templates for quick creation
- Bulk operations (create multiple shows)
- Show analytics and viewership tracking

### 6.2 Phase 3 Features
- Calendar view UI
- Conflict detection and resolution
- Team collaboration features
- Stream automation triggers
- Multi-platform support (Twitch, Facebook)
- Zoom API integration for automatic meeting creation
- Zoom meeting recording integration

## 7. Success Metrics

### 7.1 KPIs
- Number of shows created per user
- Percentage of recurring vs one-time shows
- API endpoint response times
- User retention rate
- Show completion rate

### 7.2 Monitoring
- API endpoint monitoring
- Database query performance
- Error rates and types
- User activity patterns

## 8. Testing Requirements

### 8.1 Unit Tests
- Model validation
- Business logic for recurrence patterns
- Date/time calculations
- API input validation

### 8.2 Integration Tests
- Database operations
- API endpoint flows
- Authentication/authorization
- Pagination and filtering

### 8.3 Edge Cases
- Timezone boundaries
- DST transitions
- Leap years
- Maximum recurrence limits
- Concurrent show modifications

## 9. Documentation Requirements

### 9.1 API Documentation
- OpenAPI/Swagger specification
- Example requests and responses
- Error code reference
- Rate limiting details

### 9.2 User Documentation
- Show creation guide
- Recurrence pattern examples
- Best practices
- FAQ section

## 10. Release Plan

### 10.1 MVP (Phase 1)
- Basic CRUD operations
- Simple recurrence patterns
- API implementation
- Database schema

### 10.2 Timeline
- Week 1-2: Database schema and models
- Week 3-4: API implementation
- Week 5: Testing and documentation
- Week 6: Deployment and monitoring

## 11. Risks and Mitigation

### 11.1 Technical Risks
- **Risk**: Complex recurrence calculations
  - **Mitigation**: Use established libraries (robfig/cron)
  
- **Risk**: Timezone handling errors
  - **Mitigation**: Store all times in UTC, convert on display

- **Risk**: Performance with many shows
  - **Mitigation**: Proper indexing and query optimization

### 11.2 Business Risks
- **Risk**: YouTube API changes
  - **Mitigation**: Abstract integration layer
  
- **Risk**: User adoption
  - **Mitigation**: Simple UI/UX, good documentation