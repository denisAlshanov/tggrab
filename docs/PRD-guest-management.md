# Product Requirements Document: Guest Management System

## 1. Executive Summary

### 1.1 Purpose
This PRD defines the requirements for implementing a guest management system that allows content creators to manage their show guests, maintain contact information, and quickly access guest details for show planning and event organization.

### 1.2 Scope
The guest management system will:
- Store comprehensive guest profiles with contact information
- Provide CRUD operations for guest management
- Enable fast autocomplete search for guest selection
- Support guest notes and contact management
- Integrate with existing show and event systems

### 1.3 Business Value
- **Contact Management**: Centralized guest information storage
- **Efficiency**: Quick guest lookup and selection during show planning
- **Organization**: Structured guest data with notes and contact details
- **User Experience**: Fast autocomplete for improved workflow
- **Scalability**: Foundation for future guest-event relationship features

## 2. Problem Statement

### 2.1 Current Limitations
- No centralized guest information storage
- Manual tracking of guest contact details
- No quick search functionality for guest selection
- Difficult to maintain guest relationships and notes
- No structured way to organize guest information

### 2.2 Desired Solution
- Structured guest database with comprehensive profiles
- Fast search and autocomplete functionality
- Contact management with multiple communication channels
- Note-taking capabilities for guest interactions
- RESTful API for guest operations

## 3. User Stories

### 3.1 Guest Management
1. **As a content creator**, I want to add new guests with their contact information so that I can maintain a database of potential show participants.

2. **As a content creator**, I want to update guest information when their details change so that I always have current contact information.

3. **As a content creator**, I want to search through my guest list quickly so that I can find and select guests for upcoming shows.

4. **As a content creator**, I want to add notes about each guest so that I can remember previous interactions and preferences.

### 3.2 Frontend Integration
5. **As a frontend user**, I want autocomplete suggestions when typing guest names so that I can quickly select guests without typing full names.

6. **As a frontend user**, I want to see guest contact information and notes so that I can reach out to them effectively.

7. **As a frontend user**, I want to view a paginated list of all my guests so that I can browse and manage my guest database.

## 4. Functional Requirements

### 4.1 Data Model

#### 4.1.1 Guest Entity
```go
type Guest struct {
    ID           uuid.UUID              `json:"id" db:"id"`
    UserID       uuid.UUID              `json:"user_id" db:"user_id"`
    Name         string                 `json:"name" db:"name"`
    Surname      string                 `json:"surname" db:"surname"`
    ShortName    *string                `json:"short_name,omitempty" db:"short_name"`
    Contacts     []GuestContact         `json:"contacts,omitempty" db:"contacts"`
    Notes        *string                `json:"notes,omitempty" db:"notes"`
    Avatar       *string                `json:"avatar,omitempty" db:"avatar"`
    Tags         []string               `json:"tags,omitempty" db:"tags"`
    Metadata     map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
    CreatedAt    time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

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
    Type        ContactType `json:"type" db:"type"`
    Value       string      `json:"value" db:"value"`
    Label       *string     `json:"label,omitempty" db:"label"`
    IsPrimary   bool        `json:"is_primary" db:"is_primary"`
}
```

### 4.2 Database Schema

#### 4.2.1 Guests Table
```sql
-- Create contact type enum
CREATE TYPE contact_type AS ENUM (
    'email', 'phone', 'telegram', 'discord', 'twitter', 
    'linkedin', 'instagram', 'website', 'other'
);

-- Create guests table
CREATE TABLE guests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    surname VARCHAR(255) NOT NULL,
    short_name VARCHAR(100),
    contacts JSONB DEFAULT '[]'::jsonb,
    notes TEXT,
    avatar VARCHAR(500),
    tags JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT valid_name CHECK (LENGTH(TRIM(name)) > 0),
    CONSTRAINT valid_surname CHECK (LENGTH(TRIM(surname)) > 0),
    CONSTRAINT unique_user_guest UNIQUE(user_id, name, surname)
);

-- Create indexes for performance
CREATE INDEX idx_guests_user_id ON guests(user_id);
CREATE INDEX idx_guests_name ON guests(LOWER(name));
CREATE INDEX idx_guests_surname ON guests(LOWER(surname));
CREATE INDEX idx_guests_short_name ON guests(LOWER(short_name)) WHERE short_name IS NOT NULL;
CREATE INDEX idx_guests_full_name ON guests(LOWER(name || ' ' || surname));
CREATE INDEX idx_guests_contacts ON guests USING GIN (contacts);
CREATE INDEX idx_guests_tags ON guests USING GIN (tags);
CREATE INDEX idx_guests_search ON guests USING GIN (
    (LOWER(name) || ' ' || LOWER(surname) || ' ' || COALESCE(LOWER(short_name), '')) gin_trgm_ops
);

-- Create update trigger
CREATE TRIGGER update_guests_updated_at 
BEFORE UPDATE ON guests
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();
```

### 4.3 API Endpoints

#### 4.3.1 Create Guest
**Endpoint**: `POST /api/v1/guest/new`

**Request Body**:
```json
{
    "name": "John",
    "surname": "Doe",
    "short_name": "JD",
    "contacts": [
        {
            "type": "email",
            "value": "john.doe@example.com",
            "label": "Work Email",
            "is_primary": true
        },
        {
            "type": "telegram",
            "value": "@johndoe",
            "label": "Personal",
            "is_primary": false
        }
    ],
    "notes": "Expert in AI and machine learning. Great for technical episodes.",
    "tags": ["AI", "Technical", "Expert"],
    "metadata": {
        "timezone": "America/New_York",
        "preferred_time": "afternoon"
    }
}
```

**Response**:
```json
{
    "success": true,
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "user_id": "660e8400-e29b-41d4-a716-446655440001",
        "name": "John",
        "surname": "Doe",
        "short_name": "JD",
        "contacts": [...],
        "notes": "Expert in AI and machine learning...",
        "tags": ["AI", "Technical", "Expert"],
        "created_at": "2025-01-07T10:00:00Z",
        "updated_at": "2025-01-07T10:00:00Z"
    }
}
```

#### 4.3.2 Update Guest
**Endpoint**: `PUT /api/v1/guest/update`

**Request Body**:
```json
{
    "guest_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "John",
    "surname": "Doe",
    "short_name": "Johnny",
    "contacts": [
        {
            "type": "email",
            "value": "john.new@example.com",
            "label": "New Work Email",
            "is_primary": true
        }
    ],
    "notes": "Updated notes about the guest",
    "tags": ["AI", "Technical", "Expert", "Podcaster"]
}
```

**Response**:
```json
{
    "success": true,
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "John",
        "surname": "Doe",
        "short_name": "Johnny",
        "contacts": [...],
        "notes": "Updated notes about the guest",
        "tags": ["AI", "Technical", "Expert", "Podcaster"],
        "updated_at": "2025-01-07T11:00:00Z"
    }
}
```

#### 4.3.3 List Guests
**Endpoint**: `POST /api/v1/guest/list`

**Request Body**:
```json
{
    "filters": {
        "search": "john",
        "tags": ["AI"],
        "has_contact_type": ["email"]
    },
    "pagination": {
        "page": 1,
        "limit": 20
    },
    "sort": {
        "field": "name",
        "order": "asc"
    }
}
```

**Response**:
```json
{
    "success": true,
    "data": {
        "guests": [
            {
                "id": "550e8400-e29b-41d4-a716-446655440000",
                "name": "John",
                "surname": "Doe",
                "short_name": "JD",
                "primary_email": "john.doe@example.com",
                "tags": ["AI", "Technical"],
                "notes_preview": "Expert in AI and machine learning...",
                "contact_count": 3,
                "created_at": "2025-01-07T10:00:00Z"
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

#### 4.3.4 Guest Autocomplete
**Endpoint**: `GET /api/v1/guest/autocomplete`

**Query Parameters**:
- `q` (string, required): Search query (minimum 2 characters)
- `limit` (int, optional): Maximum results (default: 10, max: 50)

**Request**: `GET /api/v1/guest/autocomplete?q=joh&limit=5`

**Response**:
```json
{
    "success": true,
    "data": {
        "suggestions": [
            {
                "id": "550e8400-e29b-41d4-a716-446655440000",
                "display_name": "John Doe (JD)",
                "name": "John",
                "surname": "Doe",
                "short_name": "JD",
                "avatar": "https://example.com/avatar.jpg",
                "primary_contact": {
                    "type": "email",
                    "value": "john.doe@example.com"
                },
                "tags": ["AI", "Technical"],
                "match_score": 0.95
            },
            {
                "id": "660e8400-e29b-41d4-a716-446655440002",
                "display_name": "Johnny Smith",
                "name": "Johnny",
                "surname": "Smith",
                "short_name": null,
                "avatar": null,
                "primary_contact": {
                    "type": "phone",
                    "value": "+1234567890"
                },
                "tags": ["Business"],
                "match_score": 0.87
            }
        ],
        "query": "joh",
        "total_matches": 2
    }
}
```

#### 4.3.5 Get Guest Details
**Endpoint**: `GET /api/v1/guest/info/{guest_id}`

**Response**:
```json
{
    "success": true,
    "data": {
        "guest": {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "name": "John",
            "surname": "Doe",
            "short_name": "JD",
            "contacts": [
                {
                    "type": "email",
                    "value": "john.doe@example.com",
                    "label": "Work Email",
                    "is_primary": true
                },
                {
                    "type": "telegram",
                    "value": "@johndoe",
                    "label": "Personal",
                    "is_primary": false
                }
            ],
            "notes": "Expert in AI and machine learning. Great for technical episodes.",
            "avatar": "https://example.com/avatar.jpg",
            "tags": ["AI", "Technical", "Expert"],
            "metadata": {
                "timezone": "America/New_York",
                "preferred_time": "afternoon"
            },
            "created_at": "2025-01-07T10:00:00Z",
            "updated_at": "2025-01-07T10:00:00Z"
        },
        "stats": {
            "total_shows": 5,
            "last_appearance": "2024-12-15T14:00:00Z",
            "upcoming_shows": 2
        }
    }
}
```

#### 4.3.6 Delete Guest
**Endpoint**: `DELETE /api/v1/guest/delete`

**Request Body**:
```json
{
    "guest_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response**:
```json
{
    "success": true,
    "message": "Guest deleted successfully",
    "data": {
        "guest_id": "550e8400-e29b-41d4-a716-446655440000",
        "deleted_at": "2025-01-07T12:00:00Z"
    }
}
```

## 5. Database Operations

### 5.1 Core CRUD Operations

#### 5.1.1 Create Guest
```go
func (p *PostgresDB) CreateGuest(ctx context.Context, guest *models.Guest) error {
    guest.ID = uuid.New()
    guest.CreatedAt = time.Now()
    guest.UpdatedAt = time.Now()

    // Convert contacts and tags to JSON
    contactsJSON, err := json.Marshal(guest.Contacts)
    if err != nil {
        return fmt.Errorf("failed to marshal contacts: %w", err)
    }

    tagsJSON, err := json.Marshal(guest.Tags)
    if err != nil {
        return fmt.Errorf("failed to marshal tags: %w", err)
    }

    metadataJSON, err := json.Marshal(guest.Metadata)
    if err != nil {
        return fmt.Errorf("failed to marshal metadata: %w", err)
    }

    query := `
        INSERT INTO guests (id, user_id, name, surname, short_name, contacts, 
            notes, avatar, tags, metadata, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        RETURNING id, created_at, updated_at`

    err = p.pool.QueryRow(ctx, query,
        guest.ID, guest.UserID, guest.Name, guest.Surname, guest.ShortName,
        contactsJSON, guest.Notes, guest.Avatar, tagsJSON, metadataJSON,
        guest.CreatedAt, guest.UpdatedAt,
    ).Scan(&guest.ID, &guest.CreatedAt, &guest.UpdatedAt)

    return err
}
```

#### 5.1.2 Search and Autocomplete
```go
func (p *PostgresDB) SearchGuests(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.GuestSuggestion, error) {
    searchQuery := `
        SELECT g.id, g.name, g.surname, g.short_name, g.avatar, g.contacts, g.tags,
               similarity(LOWER(g.name || ' ' || g.surname || ' ' || COALESCE(g.short_name, '')), LOWER($2)) as score
        FROM guests g
        WHERE g.user_id = $1 
        AND (
            LOWER(g.name) LIKE LOWER($2 || '%') OR
            LOWER(g.surname) LIKE LOWER($2 || '%') OR
            LOWER(g.short_name) LIKE LOWER($2 || '%') OR
            LOWER(g.name || ' ' || g.surname) LIKE LOWER('%' || $2 || '%')
        )
        ORDER BY score DESC, g.name ASC, g.surname ASC
        LIMIT $3`

    rows, err := p.pool.Query(ctx, searchQuery, userID, query, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var suggestions []models.GuestSuggestion
    for rows.Next() {
        var suggestion models.GuestSuggestion
        var contactsJSON, tagsJSON []byte
        var score float64

        err := rows.Scan(
            &suggestion.ID, &suggestion.Name, &suggestion.Surname, &suggestion.ShortName,
            &suggestion.Avatar, &contactsJSON, &tagsJSON, &score,
        )
        if err != nil {
            return nil, err
        }

        // Parse contacts and tags
        json.Unmarshal(contactsJSON, &suggestion.Contacts)
        json.Unmarshal(tagsJSON, &suggestion.Tags)
        
        suggestion.MatchScore = score
        suggestions = append(suggestions, suggestion)
    }

    return suggestions, nil
}
```

### 5.2 Advanced Search Features

#### 5.2.1 Full-Text Search with PostgreSQL
```sql
-- Add full-text search capability
ALTER TABLE guests ADD COLUMN search_vector tsvector;

-- Create function to update search vector
CREATE OR REPLACE FUNCTION guests_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.surname, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.short_name, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.notes, '')), 'D');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic search vector updates
CREATE TRIGGER guests_search_vector_trigger
BEFORE INSERT OR UPDATE ON guests
FOR EACH ROW EXECUTE FUNCTION guests_search_vector_update();

-- Create GIN index for fast full-text search
CREATE INDEX idx_guests_search_vector ON guests USING GIN(search_vector);
```

## 6. Request/Response Models

### 6.1 API Request Types
```go
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
```

### 6.2 API Response Types
```go
type CreateGuestResponse struct {
    Success bool   `json:"success"`
    Data    *Guest `json:"data,omitempty"`
    Error   string `json:"error,omitempty"`
}

type UpdateGuestResponse struct {
    Success bool   `json:"success"`
    Data    *Guest `json:"data,omitempty"`
    Error   string `json:"error,omitempty"`
}

type ListGuestsResponse struct {
    Success bool           `json:"success"`
    Data    *ListGuestsData `json:"data,omitempty"`
    Error   string         `json:"error,omitempty"`
}

type ListGuestsData struct {
    Guests     []GuestListItem      `json:"guests"`
    Pagination PaginationResponse   `json:"pagination"`
}

type GuestListItem struct {
    ID             uuid.UUID `json:"id"`
    Name           string    `json:"name"`
    Surname        string    `json:"surname"`
    ShortName      *string   `json:"short_name,omitempty"`
    PrimaryEmail   *string   `json:"primary_email,omitempty"`
    Avatar         *string   `json:"avatar,omitempty"`
    Tags           []string  `json:"tags"`
    NotesPreview   *string   `json:"notes_preview,omitempty"`
    ContactCount   int       `json:"contact_count"`
    CreatedAt      time.Time `json:"created_at"`
}

type AutocompleteResponse struct {
    Success bool                  `json:"success"`
    Data    *AutocompleteData     `json:"data,omitempty"`
    Error   string                `json:"error,omitempty"`
}

type AutocompleteData struct {
    Suggestions   []GuestSuggestion `json:"suggestions"`
    Query         string            `json:"query"`
    TotalMatches  int               `json:"total_matches"`
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

type DeleteGuestResponse struct {
    Success bool              `json:"success"`
    Message string            `json:"message,omitempty"`
    Data    *GuestDeleteData  `json:"data,omitempty"`
    Error   string            `json:"error,omitempty"`
}

type GuestDeleteData struct {
    GuestID   string    `json:"guest_id"`
    DeletedAt time.Time `json:"deleted_at"`
}
```

## 7. Frontend Integration

### 7.1 Autocomplete Component Design

#### 7.1.1 Search Behavior
- **Minimum Query Length**: 2 characters
- **Debounce Delay**: 300ms to avoid excessive API calls
- **Max Results**: 10 suggestions for optimal UX
- **Search Fields**: Name, Surname, Short Name
- **Fuzzy Matching**: PostgreSQL similarity scoring

#### 7.1.2 Display Format
```typescript
interface GuestSuggestion {
    id: string;
    displayName: string; // "John Doe (JD)" or "John Doe"
    avatar?: string;
    primaryContact?: {
        type: string;
        value: string;
    };
    tags: string[];
    matchScore: number;
}
```

#### 7.1.3 JavaScript Integration Example
```javascript
// Autocomplete search function
async function searchGuests(query) {
    if (query.length < 2) return [];
    
    const response = await fetch(`/api/v1/guest/autocomplete?q=${encodeURIComponent(query)}&limit=10`, {
        headers: {
            'X-API-Key': getApiKey(),
            'Content-Type': 'application/json'
        }
    });
    
    const data = await response.json();
    return data.success ? data.data.suggestions : [];
}

// Debounced search implementation
const debouncedSearch = debounce(searchGuests, 300);
```

### 7.2 Guest Management Interface

#### 7.2.1 Guest List View
- **Pagination**: 20 guests per page
- **Search**: Real-time filtering by name/surname
- **Sorting**: By name, surname, creation date
- **Filtering**: By tags, contact types
- **Actions**: Edit, Delete, View Details

#### 7.2.2 Guest Form Fields
```typescript
interface GuestForm {
    name: string;           // Required
    surname: string;        // Required
    shortName?: string;     // Optional
    contacts: Contact[];    // Dynamic list
    notes?: string;         // Textarea
    avatar?: string;        // URL or file upload
    tags: string[];         // Tag input with autocomplete
}

interface Contact {
    type: ContactType;      // Dropdown selection
    value: string;          // Input field
    label?: string;         // Optional label
    isPrimary: boolean;     // Radio/checkbox
}
```

## 8. Performance Considerations

### 8.1 Database Optimization
- **Indexing Strategy**: Composite indexes for name/surname combinations
- **Full-Text Search**: PostgreSQL tsvector for complex queries
- **JSONB Indexing**: GIN indexes for contacts and tags
- **Query Optimization**: Limit result sets and use pagination

### 8.2 Caching Strategy
```go
type GuestCache struct {
    AutocompleteCache map[string][]GuestSuggestion `json:"autocomplete"`
    RecentSearches    []string                     `json:"recent_searches"`
    TTL               time.Duration                `json:"ttl"`
}

// Cache autocomplete results for 5 minutes
const AutocompleteCacheTTL = 5 * time.Minute
```

### 8.3 API Rate Limiting
- **Autocomplete**: 100 requests per minute per user
- **CRUD Operations**: 60 requests per minute per user
- **List Operations**: 30 requests per minute per user

## 9. Security Considerations

### 9.1 Data Protection
- **User Isolation**: All guests belong to authenticated users
- **Input Validation**: Sanitize all user inputs
- **SQL Injection**: Use parameterized queries
- **XSS Prevention**: Escape output data

### 9.2 Access Control
```go
func (h *GuestHandler) validateGuestAccess(ctx context.Context, guestID uuid.UUID, userID uuid.UUID) error {
    guest, err := h.db.GetGuestByID(ctx, guestID)
    if err != nil {
        return err
    }
    
    if guest == nil {
        return NewNotFoundError("Guest not found")
    }
    
    if guest.UserID != userID {
        return NewAuthError("Access denied")
    }
    
    return nil
}
```

### 9.3 Contact Information Privacy
- **Encryption**: Sensitive contact data encryption at rest
- **Audit Logging**: Track access to guest information
- **Data Retention**: Configurable retention policies
- **Export/Import**: GDPR-compliant data export

## 10. Migration Strategy

### 10.1 Database Migration
```sql
-- Migration Version 8: Add guests table and related functionality
DO $$ BEGIN
    CREATE TYPE contact_type AS ENUM (
        'email', 'phone', 'telegram', 'discord', 'twitter', 
        'linkedin', 'instagram', 'website', 'other'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create guests table with full schema
CREATE TABLE IF NOT EXISTS guests (
    -- Full table definition as specified above
);

-- Add trigram extension for fuzzy search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Create all necessary indexes
-- (As specified in database schema section)
```

### 10.2 Data Import/Export
```go
// Import guests from CSV/JSON
func ImportGuests(ctx context.Context, userID uuid.UUID, data []GuestImport) error {
    // Validation and bulk insert logic
}

// Export guests to various formats
func ExportGuests(ctx context.Context, userID uuid.UUID, format string) ([]byte, error) {
    // Export logic for CSV, JSON, vCard formats
}
```

## 11. Testing Requirements

### 11.1 Unit Tests
- Guest CRUD operations
- Search and autocomplete algorithms
- Contact validation logic
- Permission checking

### 11.2 Integration Tests
- End-to-end API workflows
- Database constraint validation
- Search performance testing
- Concurrent user operations

### 11.3 Performance Tests
- Autocomplete response times (<100ms)
- Large dataset search performance
- Concurrent user load testing
- Database query optimization

## 12. Error Handling

### 12.1 Validation Errors
```go
type GuestValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code"`
}

const (
    ErrGuestNameRequired     = "GUEST_NAME_REQUIRED"
    ErrGuestSurnameRequired  = "GUEST_SURNAME_REQUIRED"
    ErrGuestDuplicate        = "GUEST_DUPLICATE"
    ErrInvalidContactType    = "INVALID_CONTACT_TYPE"
    ErrInvalidContactValue   = "INVALID_CONTACT_VALUE"
)
```

### 12.2 Common Error Responses
```json
{
    "success": false,
    "error": "Validation failed",
    "details": {
        "validation_errors": [
            {
                "field": "name",
                "message": "Name is required",
                "code": "GUEST_NAME_REQUIRED"
            }
        ]
    }
}
```

## 13. Monitoring & Analytics

### 13.1 Metrics to Track
- Guest creation/update/deletion rates
- Autocomplete usage and performance
- Search query patterns
- Contact type distribution
- Tag usage statistics

### 13.2 Performance Monitoring
```go
type GuestMetrics struct {
    TotalGuests        int           `json:"total_guests"`
    GuestsPerUser      float64       `json:"avg_guests_per_user"`
    AutocompleteQPS    float64       `json:"autocomplete_qps"`
    SearchLatency      time.Duration `json:"avg_search_latency"`
    MostUsedTags       []string      `json:"most_used_tags"`
    ContactTypeStats   map[string]int `json:"contact_type_stats"`
}
```

## 14. Future Enhancements

### 14.1 Phase 2 Features
- **Guest-Event Relationships**: Link guests to specific events
- **Guest Availability**: Calendar integration for scheduling
- **Communication History**: Track interactions and outreach
- **Social Media Integration**: Auto-populate from social profiles

### 14.2 Phase 3 Features
- **AI-Powered Suggestions**: Guest recommendations based on show topics
- **Integration with CRM**: Connect with external contact management
- **Bulk Operations**: Mass import/export and bulk updates
- **Advanced Analytics**: Guest engagement and appearance metrics

### 14.3 Advanced Search Features
- **Semantic Search**: Natural language guest queries
- **Relationship Mapping**: Find guests through mutual connections
- **Skill-Based Search**: Find guests by expertise areas
- **Availability-Based Search**: Find available guests for date ranges

## 15. Success Metrics

### 15.1 Technical Metrics
- **API Performance**: <100ms autocomplete response time
- **Search Accuracy**: >95% relevant results in top 5 suggestions
- **Data Consistency**: 100% data integrity across operations
- **System Reliability**: 99.9% uptime for guest services

### 15.2 User Experience Metrics
- **Adoption Rate**: Percentage of users actively managing guests
- **Search Usage**: Daily autocomplete queries per active user
- **Data Quality**: Percentage of guests with complete profiles
- **User Satisfaction**: Guest management feature satisfaction scores

## 16. Implementation Timeline

### 16.1 Phase 1 (Week 1-2): Core Foundation
- Database schema design and migration
- Basic guest data models
- Core CRUD operations
- Basic validation logic

### 16.2 Phase 2 (Week 3-4): API Development
- Guest management endpoints
- Search and autocomplete functionality
- Input validation and error handling
- Permission and security implementation

### 16.3 Phase 3 (Week 5-6): Optimization & Polish
- Performance optimization and indexing
- Advanced search features
- Comprehensive testing
- Documentation and monitoring

## 17. Risk Assessment

### 17.1 Technical Risks
- **Performance**: Large guest databases might impact search speed
  - *Mitigation*: Implement efficient indexing and caching strategies
- **Data Quality**: Inconsistent contact information formats
  - *Mitigation*: Strong validation and normalization rules
- **Scalability**: Search performance with thousands of guests
  - *Mitigation*: Database optimization and search result limits

### 17.2 User Experience Risks
- **Search Confusion**: Users might not find expected guests
  - *Mitigation*: Implement fuzzy matching and multiple search strategies
- **Data Entry Burden**: Complex guest forms might discourage usage
  - *Mitigation*: Progressive disclosure and smart defaults
- **Performance Expectations**: Users expect instant autocomplete
  - *Mitigation*: Optimize for <100ms response times with caching