# Product Requirements Document: Show Blocks System

## 1. Executive Summary

### 1.1 Purpose
This PRD defines the requirements for implementing a show blocks system that allows content creators to structure their shows and events with modular, reusable content blocks. Each block represents a segment of the show with its own topic, duration, guests, and media resources.

### 1.2 Scope
The show blocks system will:
- Enable creation of structured show segments (blocks)
- Support block ordering and reordering within events
- Associate guests and media with specific blocks
- Provide estimated timing for show planning
- Allow reuse of blocks across different events

### 1.3 Business Value
- **Content Organization**: Structure shows into logical segments
- **Time Management**: Better planning with block duration estimates
- **Resource Association**: Link guests and media to specific segments
- **Flexibility**: Reorder blocks dynamically for optimal flow
- **Reusability**: Use successful blocks across multiple shows
- **Production Planning**: Clear segment breakdown for production teams

## 2. Problem Statement

### 2.1 Current Limitations
- No way to structure show content into segments
- Difficult to estimate show timing by segments
- Cannot associate guests with specific parts of shows
- No ability to reuse successful show segments
- Hard to reorganize show flow dynamically
- Limited production planning capabilities

### 2.2 Desired Solution
- Modular block system for show structuring
- Drag-and-drop reordering capabilities
- Guest and media association per block
- Time estimation and tracking
- Block templates for reuse
- Clear API for block management

## 3. User Stories

### 3.1 Block Creation and Management
1. **As a content creator**, I want to create blocks with topics and descriptions so that I can structure my show content.

2. **As a content creator**, I want to estimate block duration so that I can plan my show timing accurately.

3. **As a content creator**, I want to update block details so that I can refine content as needed.

4. **As a content creator**, I want to delete blocks so that I can remove unwanted segments.

### 3.2 Block Organization
5. **As a content creator**, I want to reorder blocks within an event so that I can optimize show flow.

6. **As a content creator**, I want to see all blocks in an event in order so that I understand the show structure.

### 3.3 Resource Association
7. **As a content creator**, I want to assign guests to specific blocks so that I know who participates in which segments.

8. **As a content creator**, I want to attach media to blocks so that I have resources ready for each segment.

### 3.4 Production Planning
9. **As a production team member**, I want to see block details with timing so that I can prepare for each segment.

10. **As a content creator**, I want to reuse successful blocks in new events so that I can maintain consistency.

## 4. Functional Requirements

### 4.1 Data Model

#### 4.1.1 Block Entity
```go
type Block struct {
    ID              uuid.UUID              `json:"id" db:"id"`
    EventID         uuid.UUID              `json:"event_id" db:"event_id"`
    UserID          uuid.UUID              `json:"user_id" db:"user_id"`
    Title           string                 `json:"title" db:"title"`
    Description     *string                `json:"description,omitempty" db:"description"`
    Topic           *string                `json:"topic,omitempty" db:"topic"`
    EstimatedLength int                    `json:"estimated_length" db:"estimated_length"` // in minutes
    ActualLength    *int                   `json:"actual_length,omitempty" db:"actual_length"`
    OrderIndex      int                    `json:"order_index" db:"order_index"`
    BlockType       BlockType              `json:"block_type" db:"block_type"`
    Status          BlockStatus            `json:"status" db:"status"`
    Metadata        map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
    CreatedAt       time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

type BlockType string

const (
    BlockTypeIntro      BlockType = "intro"
    BlockTypeMain       BlockType = "main"
    BlockTypeInterview  BlockType = "interview"
    BlockTypeQA         BlockType = "qa"
    BlockTypeBreak      BlockType = "break"
    BlockTypeOutro      BlockType = "outro"
    BlockTypeCustom     BlockType = "custom"
)

type BlockStatus string

const (
    BlockStatusPlanned    BlockStatus = "planned"
    BlockStatusReady      BlockStatus = "ready"
    BlockStatusInProgress BlockStatus = "in_progress"
    BlockStatusCompleted  BlockStatus = "completed"
    BlockStatusSkipped    BlockStatus = "skipped"
)

// Junction table for block-guest relationships
type BlockGuest struct {
    ID        uuid.UUID   `json:"id" db:"id"`
    BlockID   uuid.UUID   `json:"block_id" db:"block_id"`
    GuestID   uuid.UUID   `json:"guest_id" db:"guest_id"`
    Role      *string     `json:"role,omitempty" db:"role"`
    Notes     *string     `json:"notes,omitempty" db:"notes"`
    CreatedAt time.Time   `json:"created_at" db:"created_at"`
}

// Junction table for block-media relationships
type BlockMedia struct {
    ID          uuid.UUID   `json:"id" db:"id"`
    BlockID     uuid.UUID   `json:"block_id" db:"block_id"`
    MediaID     uuid.UUID   `json:"media_id" db:"media_id"`
    MediaType   string      `json:"media_type" db:"media_type"`
    Title       *string     `json:"title,omitempty" db:"title"`
    Description *string     `json:"description,omitempty" db:"description"`
    OrderIndex  int         `json:"order_index" db:"order_index"`
    CreatedAt   time.Time   `json:"created_at" db:"created_at"`
}
```

### 4.2 Database Schema

#### 4.2.1 Blocks Table
```sql
-- Create block type enum
CREATE TYPE block_type AS ENUM (
    'intro', 'main', 'interview', 'qa', 'break', 'outro', 'custom'
);

-- Create block status enum
CREATE TYPE block_status AS ENUM (
    'planned', 'ready', 'in_progress', 'completed', 'skipped'
);

-- Create blocks table
CREATE TABLE blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    topic VARCHAR(500),
    estimated_length INTEGER NOT NULL DEFAULT 5, -- minutes
    actual_length INTEGER,
    order_index INTEGER NOT NULL,
    block_type block_type DEFAULT 'custom',
    status block_status DEFAULT 'planned',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT valid_title CHECK (LENGTH(TRIM(title)) > 0),
    CONSTRAINT positive_estimated_length CHECK (estimated_length > 0),
    CONSTRAINT positive_actual_length CHECK (actual_length IS NULL OR actual_length > 0),
    CONSTRAINT unique_event_order UNIQUE(event_id, order_index)
);

-- Create indexes
CREATE INDEX idx_blocks_event_id ON blocks(event_id);
CREATE INDEX idx_blocks_user_id ON blocks(user_id);
CREATE INDEX idx_blocks_event_order ON blocks(event_id, order_index);
CREATE INDEX idx_blocks_status ON blocks(status);
CREATE INDEX idx_blocks_type ON blocks(block_type);

-- Create block_guests junction table
CREATE TABLE block_guests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    block_id UUID NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
    guest_id UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    role VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT unique_block_guest UNIQUE(block_id, guest_id)
);

-- Create indexes for block_guests
CREATE INDEX idx_block_guests_block_id ON block_guests(block_id);
CREATE INDEX idx_block_guests_guest_id ON block_guests(guest_id);

-- Create block_media junction table
CREATE TABLE block_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    block_id UUID NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    media_type VARCHAR(50) NOT NULL,
    title VARCHAR(255),
    description TEXT,
    order_index INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT unique_block_media_order UNIQUE(block_id, order_index)
);

-- Create indexes for block_media
CREATE INDEX idx_block_media_block_id ON block_media(block_id);
CREATE INDEX idx_block_media_media_id ON block_media(media_id);
CREATE INDEX idx_block_media_order ON block_media(block_id, order_index);

-- Create update triggers
CREATE TRIGGER update_blocks_updated_at 
BEFORE UPDATE ON blocks
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();
```

### 4.3 API Endpoints

#### 4.3.1 Add Block
**Endpoint**: `POST /api/v1/block/add`

**Request Body**:
```json
{
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Opening Segment",
    "description": "Welcome viewers and introduce today's topics",
    "topic": "AI in Content Creation",
    "estimated_length": 10,
    "block_type": "intro",
    "order_index": 1,
    "guest_ids": ["660e8400-e29b-41d4-a716-446655440001"],
    "media": [
        {
            "media_id": "770e8400-e29b-41d4-a716-446655440002",
            "media_type": "image",
            "title": "Show Banner",
            "order_index": 0
        }
    ],
    "metadata": {
        "talking_points": ["Welcome", "Today's agenda", "Guest introduction"],
        "notes": "Remember to mention sponsor"
    }
}
```

**Response**:
```json
{
    "success": true,
    "data": {
        "id": "880e8400-e29b-41d4-a716-446655440003",
        "event_id": "550e8400-e29b-41d4-a716-446655440000",
        "user_id": "990e8400-e29b-41d4-a716-446655440004",
        "title": "Opening Segment",
        "description": "Welcome viewers and introduce today's topics",
        "topic": "AI in Content Creation",
        "estimated_length": 10,
        "actual_length": null,
        "order_index": 1,
        "block_type": "intro",
        "status": "planned",
        "guests": [
            {
                "id": "660e8400-e29b-41d4-a716-446655440001",
                "name": "John",
                "surname": "Doe",
                "role": "Expert Guest"
            }
        ],
        "media": [
            {
                "media_id": "770e8400-e29b-41d4-a716-446655440002",
                "media_type": "image",
                "title": "Show Banner",
                "order_index": 0
            }
        ],
        "metadata": {
            "talking_points": ["Welcome", "Today's agenda", "Guest introduction"],
            "notes": "Remember to mention sponsor"
        },
        "created_at": "2025-01-07T10:00:00Z",
        "updated_at": "2025-01-07T10:00:00Z"
    }
}
```

#### 4.3.2 Update Block
**Endpoint**: `PUT /api/v1/block/update`

**Request Body**:
```json
{
    "block_id": "880e8400-e29b-41d4-a716-446655440003",
    "title": "Extended Opening Segment",
    "description": "Welcome viewers with extended introduction",
    "estimated_length": 15,
    "actual_length": 12,
    "status": "completed",
    "guest_ids": ["660e8400-e29b-41d4-a716-446655440001", "aa0e8400-e29b-41d4-a716-446655440005"],
    "media": [
        {
            "media_id": "770e8400-e29b-41d4-a716-446655440002",
            "media_type": "image",
            "title": "Updated Show Banner",
            "order_index": 0
        },
        {
            "media_id": "bb0e8400-e29b-41d4-a716-446655440006",
            "media_type": "video",
            "title": "Intro Animation",
            "order_index": 1
        }
    ]
}
```

**Response**:
```json
{
    "success": true,
    "data": {
        "id": "880e8400-e29b-41d4-a716-446655440003",
        "title": "Extended Opening Segment",
        "estimated_length": 15,
        "actual_length": 12,
        "status": "completed",
        "guests": [...],
        "media": [...],
        "updated_at": "2025-01-07T11:00:00Z"
    }
}
```

#### 4.3.3 Get Block Info
**Endpoint**: `GET /api/v1/block/info/{block_id}`

**Response**:
```json
{
    "success": true,
    "data": {
        "block": {
            "id": "880e8400-e29b-41d4-a716-446655440003",
            "event_id": "550e8400-e29b-41d4-a716-446655440000",
            "title": "Opening Segment",
            "description": "Welcome viewers and introduce today's topics",
            "topic": "AI in Content Creation",
            "estimated_length": 10,
            "actual_length": null,
            "order_index": 1,
            "block_type": "intro",
            "status": "planned",
            "guests": [
                {
                    "id": "660e8400-e29b-41d4-a716-446655440001",
                    "name": "John",
                    "surname": "Doe",
                    "short_name": "JD",
                    "role": "Expert Guest",
                    "primary_contact": {
                        "type": "email",
                        "value": "john.doe@example.com"
                    }
                }
            ],
            "media": [
                {
                    "media_id": "770e8400-e29b-41d4-a716-446655440002",
                    "media_type": "image",
                    "title": "Show Banner",
                    "file_name": "banner.png",
                    "file_size": 1048576,
                    "s3_url": "https://s3.example.com/media/banner.png"
                }
            ],
            "created_at": "2025-01-07T10:00:00Z",
            "updated_at": "2025-01-07T10:00:00Z"
        },
        "event_info": {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "show_name": "Tech Talk Tuesday",
            "start_datetime": "2025-01-14T15:00:00Z",
            "total_blocks": 5,
            "total_estimated_time": 60
        }
    }
}
```

#### 4.3.4 Reorder Blocks
**Endpoint**: `PUT /api/v1/block/reorder`

**Request Body**:
```json
{
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "block_orders": [
        {
            "block_id": "880e8400-e29b-41d4-a716-446655440003",
            "order_index": 2
        },
        {
            "block_id": "990e8400-e29b-41d4-a716-446655440004",
            "order_index": 1
        },
        {
            "block_id": "aa0e8400-e29b-41d4-a716-446655440005",
            "order_index": 3
        }
    ]
}
```

**Response**:
```json
{
    "success": true,
    "data": {
        "event_id": "550e8400-e29b-41d4-a716-446655440000",
        "blocks": [
            {
                "block_id": "990e8400-e29b-41d4-a716-446655440004",
                "title": "Guest Introduction",
                "order_index": 1
            },
            {
                "block_id": "880e8400-e29b-41d4-a716-446655440003",
                "title": "Opening Segment",
                "order_index": 2
            },
            {
                "block_id": "aa0e8400-e29b-41d4-a716-446655440005",
                "title": "Main Discussion",
                "order_index": 3
            }
        ],
        "total_estimated_time": 60
    }
}
```

#### 4.3.5 Delete Block
**Endpoint**: `DELETE /api/v1/block/delete`

**Request Body**:
```json
{
    "block_id": "880e8400-e29b-41d4-a716-446655440003",
    "reorder_remaining": true
}
```

**Response**:
```json
{
    "success": true,
    "message": "Block deleted successfully",
    "data": {
        "block_id": "880e8400-e29b-41d4-a716-446655440003",
        "deleted_at": "2025-01-07T12:00:00Z",
        "remaining_blocks_reordered": true
    }
}
```

#### 4.3.6 List Event Blocks
**Endpoint**: `GET /api/v1/event/{event_id}/blocks`

**Response**:
```json
{
    "success": true,
    "data": {
        "event_id": "550e8400-e29b-41d4-a716-446655440000",
        "blocks": [
            {
                "id": "990e8400-e29b-41d4-a716-446655440004",
                "title": "Guest Introduction",
                "topic": "Meet our expert",
                "estimated_length": 5,
                "order_index": 1,
                "block_type": "intro",
                "status": "ready",
                "guest_count": 1,
                "media_count": 2
            },
            {
                "id": "880e8400-e29b-41d4-a716-446655440003",
                "title": "Opening Segment",
                "topic": "AI in Content Creation",
                "estimated_length": 10,
                "order_index": 2,
                "block_type": "main",
                "status": "planned",
                "guest_count": 2,
                "media_count": 1
            }
        ],
        "total_blocks": 2,
        "total_estimated_time": 15,
        "total_actual_time": 0
    }
}
```

## 5. Database Operations

### 5.1 Core CRUD Operations

#### 5.1.1 Create Block
```go
func (p *PostgresDB) CreateBlock(ctx context.Context, block *models.Block, guestIDs []uuid.UUID, media []models.BlockMediaInput) error {
    tx, err := p.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // Insert block
    block.ID = uuid.New()
    block.CreatedAt = time.Now()
    block.UpdatedAt = time.Now()

    metadataJSON, _ := json.Marshal(block.Metadata)

    query := `
        INSERT INTO blocks (id, event_id, user_id, title, description, topic,
            estimated_length, order_index, block_type, status, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        RETURNING created_at, updated_at`

    err = tx.QueryRow(ctx, query,
        block.ID, block.EventID, block.UserID, block.Title, block.Description,
        block.Topic, block.EstimatedLength, block.OrderIndex, block.BlockType,
        block.Status, metadataJSON,
    ).Scan(&block.CreatedAt, &block.UpdatedAt)

    if err != nil {
        return err
    }

    // Insert block-guest relationships
    for _, guestID := range guestIDs {
        _, err = tx.Exec(ctx, `
            INSERT INTO block_guests (block_id, guest_id)
            VALUES ($1, $2)`, block.ID, guestID)
        if err != nil {
            return err
        }
    }

    // Insert block-media relationships
    for _, m := range media {
        _, err = tx.Exec(ctx, `
            INSERT INTO block_media (block_id, media_id, media_type, title, order_index)
            VALUES ($1, $2, $3, $4, $5)`,
            block.ID, m.MediaID, m.MediaType, m.Title, m.OrderIndex)
        if err != nil {
            return err
        }
    }

    return tx.Commit(ctx)
}
```

#### 5.1.2 Reorder Blocks
```go
func (p *PostgresDB) ReorderBlocks(ctx context.Context, eventID uuid.UUID, blockOrders []models.BlockOrder) error {
    tx, err := p.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // Verify all blocks belong to the event
    blockIDs := make([]uuid.UUID, len(blockOrders))
    for i, bo := range blockOrders {
        blockIDs[i] = bo.BlockID
    }

    var count int
    err = tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM blocks 
        WHERE event_id = $1 AND id = ANY($2)`,
        eventID, blockIDs).Scan(&count)
    
    if err != nil {
        return err
    }
    
    if count != len(blockOrders) {
        return fmt.Errorf("some blocks do not belong to this event")
    }

    // Update order indexes
    for _, bo := range blockOrders {
        _, err = tx.Exec(ctx, `
            UPDATE blocks SET order_index = $2 
            WHERE id = $1 AND event_id = $3`,
            bo.BlockID, bo.OrderIndex, eventID)
        if err != nil {
            return err
        }
    }

    return tx.Commit(ctx)
}
```

### 5.2 Query Operations

#### 5.2.1 Get Block with Relations
```go
func (p *PostgresDB) GetBlockWithRelations(ctx context.Context, blockID uuid.UUID) (*models.BlockDetail, error) {
    block := &models.BlockDetail{}
    
    // Get block data
    err := p.pool.QueryRow(ctx, `
        SELECT b.*, e.show_id, e.start_datetime, s.show_name
        FROM blocks b
        JOIN events e ON b.event_id = e.id
        JOIN shows s ON e.show_id = s.id
        WHERE b.id = $1`, blockID).Scan(&block.Block)
    
    if err != nil {
        return nil, err
    }

    // Get guests
    rows, err := p.pool.Query(ctx, `
        SELECT g.*, bg.role, bg.notes
        FROM guests g
        JOIN block_guests bg ON g.id = bg.guest_id
        WHERE bg.block_id = $1
        ORDER BY g.name, g.surname`, blockID)
    
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    // Process guests...

    // Get media
    mediaRows, err := p.pool.Query(ctx, `
        SELECT m.*, bm.title, bm.description, bm.order_index
        FROM media m
        JOIN block_media bm ON m.id = bm.media_id
        WHERE bm.block_id = $1
        ORDER BY bm.order_index`, blockID)
    
    // Process media...

    return block, nil
}
```

## 6. Request/Response Models

### 6.1 API Request Types
```go
type AddBlockRequest struct {
    EventID         string                 `json:"event_id" binding:"required,uuid"`
    Title           string                 `json:"title" binding:"required,min=1,max=255"`
    Description     *string                `json:"description,omitempty"`
    Topic           *string                `json:"topic,omitempty"`
    EstimatedLength int                    `json:"estimated_length" binding:"required,min=1,max=480"`
    BlockType       BlockType              `json:"block_type,omitempty"`
    OrderIndex      int                    `json:"order_index,omitempty"`
    GuestIDs        []string               `json:"guest_ids,omitempty"`
    Media           []BlockMediaInput      `json:"media,omitempty"`
    Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateBlockRequest struct {
    BlockID         string                 `json:"block_id" binding:"required,uuid"`
    Title           *string                `json:"title,omitempty"`
    Description     *string                `json:"description,omitempty"`
    Topic           *string                `json:"topic,omitempty"`
    EstimatedLength *int                   `json:"estimated_length,omitempty"`
    ActualLength    *int                   `json:"actual_length,omitempty"`
    BlockType       *BlockType             `json:"block_type,omitempty"`
    Status          *BlockStatus           `json:"status,omitempty"`
    GuestIDs        []string               `json:"guest_ids,omitempty"`
    Media           []BlockMediaInput      `json:"media,omitempty"`
    Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type ReorderBlocksRequest struct {
    EventID     string       `json:"event_id" binding:"required,uuid"`
    BlockOrders []BlockOrder `json:"block_orders" binding:"required,min=1"`
}

type BlockOrder struct {
    BlockID    string `json:"block_id" binding:"required,uuid"`
    OrderIndex int    `json:"order_index" binding:"required,min=0"`
}

type DeleteBlockRequest struct {
    BlockID          string `json:"block_id" binding:"required,uuid"`
    ReorderRemaining bool   `json:"reorder_remaining,omitempty"`
}

type BlockMediaInput struct {
    MediaID     string  `json:"media_id" binding:"required,uuid"`
    MediaType   string  `json:"media_type" binding:"required"`
    Title       *string `json:"title,omitempty"`
    Description *string `json:"description,omitempty"`
    OrderIndex  int     `json:"order_index,omitempty"`
}
```

### 6.2 API Response Types
```go
type AddBlockResponse struct {
    Success bool         `json:"success"`
    Data    *BlockDetail `json:"data,omitempty"`
    Error   string       `json:"error,omitempty"`
}

type UpdateBlockResponse struct {
    Success bool         `json:"success"`
    Data    *BlockDetail `json:"data,omitempty"`
    Error   string       `json:"error,omitempty"`
}

type GetBlockInfoResponse struct {
    Success bool             `json:"success"`
    Data    *BlockInfoData   `json:"data,omitempty"`
    Error   string           `json:"error,omitempty"`
}

type BlockInfoData struct {
    Block      *BlockDetail    `json:"block"`
    EventInfo  *EventSummary   `json:"event_info"`
}

type BlockDetail struct {
    models.Block
    Guests []BlockGuestDetail `json:"guests"`
    Media  []BlockMediaDetail `json:"media"`
}

type BlockGuestDetail struct {
    ID             uuid.UUID     `json:"id"`
    Name           string        `json:"name"`
    Surname        string        `json:"surname"`
    ShortName      *string       `json:"short_name,omitempty"`
    Role           *string       `json:"role,omitempty"`
    Notes          *string       `json:"notes,omitempty"`
    PrimaryContact *GuestContact `json:"primary_contact,omitempty"`
}

type BlockMediaDetail struct {
    MediaID     uuid.UUID `json:"media_id"`
    MediaType   string    `json:"media_type"`
    Title       *string   `json:"title,omitempty"`
    Description *string   `json:"description,omitempty"`
    FileName    string    `json:"file_name"`
    FileSize    int64     `json:"file_size"`
    S3URL       string    `json:"s3_url"`
    OrderIndex  int       `json:"order_index"`
}

type ReorderBlocksResponse struct {
    Success bool               `json:"success"`
    Data    *ReorderBlocksData `json:"data,omitempty"`
    Error   string             `json:"error,omitempty"`
}

type ReorderBlocksData struct {
    EventID             string              `json:"event_id"`
    Blocks              []BlockOrderSummary `json:"blocks"`
    TotalEstimatedTime  int                 `json:"total_estimated_time"`
}

type BlockOrderSummary struct {
    BlockID    string `json:"block_id"`
    Title      string `json:"title"`
    OrderIndex int    `json:"order_index"`
}

type DeleteBlockResponse struct {
    Success bool             `json:"success"`
    Message string           `json:"message,omitempty"`
    Data    *DeleteBlockData `json:"data,omitempty"`
    Error   string           `json:"error,omitempty"`
}

type DeleteBlockData struct {
    BlockID                 string    `json:"block_id"`
    DeletedAt               time.Time `json:"deleted_at"`
    RemainingBlocksReordered bool     `json:"remaining_blocks_reordered"`
}

type EventBlocksResponse struct {
    Success bool              `json:"success"`
    Data    *EventBlocksData  `json:"data,omitempty"`
    Error   string            `json:"error,omitempty"`
}

type EventBlocksData struct {
    EventID            string            `json:"event_id"`
    Blocks             []BlockSummary    `json:"blocks"`
    TotalBlocks        int               `json:"total_blocks"`
    TotalEstimatedTime int               `json:"total_estimated_time"`
    TotalActualTime    int               `json:"total_actual_time"`
}

type BlockSummary struct {
    ID              uuid.UUID   `json:"id"`
    Title           string      `json:"title"`
    Topic           *string     `json:"topic,omitempty"`
    EstimatedLength int         `json:"estimated_length"`
    ActualLength    *int        `json:"actual_length,omitempty"`
    OrderIndex      int         `json:"order_index"`
    BlockType       BlockType   `json:"block_type"`
    Status          BlockStatus `json:"status"`
    GuestCount      int         `json:"guest_count"`
    MediaCount      int         `json:"media_count"`
}
```

## 7. Frontend Integration

### 7.1 Block Management Interface

#### 7.1.1 Block Editor
- **Title**: Required text field (max 255 chars)
- **Description**: Optional rich text editor
- **Topic**: Optional text field for segment topic
- **Duration**: Number input (1-480 minutes)
- **Type**: Dropdown (intro, main, interview, Q&A, break, outro, custom)
- **Status**: Status indicator (planned, ready, in-progress, completed, skipped)

#### 7.1.2 Guest Assignment
```typescript
interface BlockGuestAssignment {
    blockId: string;
    guests: Array<{
        guestId: string;
        displayName: string;
        role?: string;
        notes?: string;
    }>;
}

// Autocomplete integration
async function assignGuestToBlock(blockId: string, query: string) {
    const guests = await searchGuests(query);
    // Show autocomplete dropdown
    // Allow multiple selection
    // Optional role assignment
}
```

#### 7.1.3 Media Management
```typescript
interface BlockMediaManagement {
    blockId: string;
    media: Array<{
        mediaId: string;
        mediaType: string;
        title?: string;
        thumbnail?: string;
        orderIndex: number;
    }>;
}

// Drag-and-drop media ordering
function reorderBlockMedia(blockId: string, mediaItems: MediaItem[]) {
    // Update order indexes
    // Save to backend
}
```

#### 7.1.4 Block Reordering
```typescript
// Drag-and-drop interface
interface DragDropReorder {
    eventId: string;
    blocks: BlockSummary[];
    onReorder: (newOrder: BlockOrder[]) => void;
}

// Visual timeline
interface BlockTimeline {
    blocks: Array<{
        id: string;
        title: string;
        startTime: number; // minutes from event start
        duration: number;
        type: BlockType;
        status: BlockStatus;
    }>;
    totalDuration: number;
}
```

## 8. Business Logic

### 8.1 Block Ordering Rules
```go
func ValidateBlockOrder(blocks []models.Block) error {
    // Check for gaps in order
    orderMap := make(map[int]bool)
    for _, block := range blocks {
        if orderMap[block.OrderIndex] {
            return fmt.Errorf("duplicate order index: %d", block.OrderIndex)
        }
        orderMap[block.OrderIndex] = true
    }
    
    // Ensure continuous ordering
    for i := 0; i < len(blocks); i++ {
        if !orderMap[i] {
            return fmt.Errorf("missing order index: %d", i)
        }
    }
    
    return nil
}

func RebalanceBlockOrder(ctx context.Context, eventID uuid.UUID) error {
    // Get all blocks for event
    // Reorder with continuous indexes starting from 0
    // Update database
}
```

### 8.2 Time Calculations
```go
func CalculateEventTiming(blocks []models.Block) EventTiming {
    var totalEstimated, totalActual int
    var hasActual bool
    
    for _, block := range blocks {
        totalEstimated += block.EstimatedLength
        if block.ActualLength != nil {
            totalActual += *block.ActualLength
            hasActual = true
        }
    }
    
    return EventTiming{
        TotalEstimatedMinutes: totalEstimated,
        TotalActualMinutes:    totalActual,
        HasActualTiming:       hasActual,
        EstimatedEndTime:      calculateEndTime(totalEstimated),
    }
}
```

### 8.3 Block Templates
```go
type BlockTemplate struct {
    ID              uuid.UUID              `json:"id"`
    UserID          uuid.UUID              `json:"user_id"`
    Name            string                 `json:"name"`
    Description     string                 `json:"description"`
    BlockType       BlockType              `json:"block_type"`
    EstimatedLength int                    `json:"estimated_length"`
    TemplateData    map[string]interface{} `json:"template_data"`
    UsageCount      int                    `json:"usage_count"`
    CreatedAt       time.Time              `json:"created_at"`
}

func CreateBlockFromTemplate(template BlockTemplate, eventID uuid.UUID) models.Block {
    return models.Block{
        EventID:         eventID,
        Title:           template.Name,
        Description:     &template.Description,
        EstimatedLength: template.EstimatedLength,
        BlockType:       template.BlockType,
        Metadata:        template.TemplateData,
    }
}
```

## 9. Performance Considerations

### 9.1 Database Optimization
- **Composite Indexes**: `(event_id, order_index)` for fast ordering queries
- **Junction Table Indexes**: Optimize guest and media lookups
- **Cascade Deletes**: Automatic cleanup of related data
- **Batch Operations**: Bulk insert/update for reordering

### 9.2 Query Optimization
```sql
-- Efficient block listing with counts
SELECT b.*,
    COUNT(DISTINCT bg.guest_id) as guest_count,
    COUNT(DISTINCT bm.media_id) as media_count
FROM blocks b
LEFT JOIN block_guests bg ON b.id = bg.block_id
LEFT JOIN block_media bm ON b.id = bm.block_id
WHERE b.event_id = $1
GROUP BY b.id
ORDER BY b.order_index;

-- Preload related data
WITH block_data AS (
    SELECT * FROM blocks WHERE event_id = $1
),
guest_data AS (
    SELECT bg.*, g.name, g.surname, g.short_name
    FROM block_guests bg
    JOIN guests g ON bg.guest_id = g.id
    WHERE bg.block_id IN (SELECT id FROM block_data)
),
media_data AS (
    SELECT bm.*, m.file_name, m.file_size
    FROM block_media bm
    JOIN media m ON bm.media_id = m.id
    WHERE bm.block_id IN (SELECT id FROM block_data)
)
SELECT * FROM block_data, guest_data, media_data;
```

### 9.3 Caching Strategy
```go
type BlockCache struct {
    EventBlocks map[uuid.UUID][]BlockSummary `json:"event_blocks"`
    BlockDetail map[uuid.UUID]BlockDetail     `json:"block_detail"`
    TTL         time.Duration                 `json:"ttl"`
}

// Cache event block lists for 5 minutes
// Cache individual block details for 10 minutes
// Invalidate on any block update/reorder
```

## 10. Security Considerations

### 10.1 Access Control
```go
func ValidateBlockAccess(ctx context.Context, userID, blockID uuid.UUID) error {
    // Verify block belongs to user
    var blockUserID uuid.UUID
    err := db.QueryRow(ctx, `
        SELECT user_id FROM blocks WHERE id = $1`,
        blockID).Scan(&blockUserID)
    
    if err != nil {
        return NewNotFoundError("block not found")
    }
    
    if blockUserID != userID {
        return NewAuthError("access denied")
    }
    
    return nil
}
```

### 10.2 Input Validation
- **Order Index**: Prevent negative values and duplicates
- **Duration**: Reasonable limits (1-480 minutes)
- **Guest/Media IDs**: Verify ownership before association
- **Metadata**: Size limits to prevent abuse

### 10.3 Data Integrity
- **Foreign Key Constraints**: Ensure referential integrity
- **Transaction Boundaries**: Atomic operations for complex updates
- **Cascade Rules**: Proper cleanup on deletions
- **Unique Constraints**: Prevent duplicate orders

## 11. Error Handling

### 11.1 Common Errors
```go
const (
    ErrBlockNotFound        = "BLOCK_NOT_FOUND"
    ErrInvalidOrder         = "INVALID_BLOCK_ORDER"
    ErrDuplicateOrder       = "DUPLICATE_ORDER_INDEX"
    ErrInvalidDuration      = "INVALID_DURATION"
    ErrEventNotFound        = "EVENT_NOT_FOUND"
    ErrGuestNotFound        = "GUEST_NOT_FOUND"
    ErrMediaNotFound        = "MEDIA_NOT_FOUND"
    ErrAccessDenied         = "ACCESS_DENIED"
    ErrOrderGap             = "ORDER_INDEX_GAP"
)
```

### 11.2 Error Responses
```json
{
    "success": false,
    "error": "Validation failed",
    "details": {
        "code": "DUPLICATE_ORDER_INDEX",
        "message": "Order index 2 is already used by another block",
        "field": "order_index",
        "value": 2
    }
}
```

## 12. Testing Requirements

### 12.1 Unit Tests
- Block CRUD operations
- Order validation logic
- Time calculation functions
- Access control checks

### 12.2 Integration Tests
- Full block lifecycle (create, update, reorder, delete)
- Guest and media associations
- Event timing calculations
- Concurrent reordering

### 12.3 Performance Tests
- Large event handling (50+ blocks)
- Bulk reordering operations
- Complex queries with relations
- Concurrent user operations

## 13. Migration Strategy

### 13.1 Database Migration
```sql
-- Migration Version 9: Add blocks system
DO $$ BEGIN
    -- Create enums
    CREATE TYPE block_type AS ENUM (...);
    CREATE TYPE block_status AS ENUM (...);
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create tables with all constraints and indexes
-- Add foreign key relationships
-- Create update triggers
```

### 13.2 Data Migration
- No existing data to migrate for new feature
- Consider creating default block templates
- Add sample blocks for testing

## 14. Monitoring & Analytics

### 14.1 Metrics to Track
- Average blocks per event
- Most used block types
- Average block duration accuracy
- Block reordering frequency
- Guest participation by block type

### 14.2 Performance Metrics
```go
type BlockMetrics struct {
    AverageBlocksPerEvent   float64           `json:"avg_blocks_per_event"`
    BlockTypeDistribution   map[string]int    `json:"block_type_distribution"`
    AverageDurationAccuracy float64           `json:"avg_duration_accuracy"`
    ReorderingFrequency     float64           `json:"reordering_frequency"`
    PopularTemplates        []TemplateUsage   `json:"popular_templates"`
}
```

## 15. Future Enhancements

### 15.1 Phase 2 Features
- **Block Templates Library**: Share successful blocks
- **AI Suggestions**: Recommend block order based on content
- **Time Tracking**: Real-time duration tracking during events
- **Block Analytics**: Performance metrics per block type

### 15.2 Phase 3 Features
- **Collaborative Editing**: Multiple users editing blocks
- **Version Control**: Track block changes over time
- **Block Scheduling**: Auto-generate blocks from templates
- **Integration**: Connect with streaming software

### 15.3 Advanced Features
- **Block Transitions**: Define how to move between blocks
- **Dynamic Timing**: Adjust remaining blocks based on overruns
- **Guest Scheduling**: Notify guests of their block times
- **Production Notes**: Detailed instructions per block

## 16. Success Metrics

### 16.1 Technical Metrics
- **API Performance**: <200ms response for block operations
- **Reorder Speed**: <500ms for reordering 20 blocks
- **Data Integrity**: 100% consistency in block ordering
- **System Reliability**: 99.9% uptime for block services

### 16.2 User Experience Metrics
- **Adoption Rate**: 80% of events use blocks
- **Average Blocks**: 5-7 blocks per event
- **Time Accuracy**: Within 10% of estimated duration
- **User Satisfaction**: Block feature satisfaction scores

## 17. Implementation Timeline

### 17.1 Phase 1 (Week 1-2): Core Foundation
- Database schema and migrations
- Basic block CRUD operations
- Order management logic
- Initial API endpoints

### 17.2 Phase 2 (Week 3-4): Integration
- Guest and media associations
- Event integration
- Reordering functionality
- Frontend API completion

### 17.3 Phase 3 (Week 5-6): Polish
- Performance optimization
- Advanced queries
- Error handling improvements
- Comprehensive testing

## 18. Risk Assessment

### 18.1 Technical Risks
- **Ordering Complexity**: Managing concurrent reorders
  - *Mitigation*: Database transactions and locks
- **Performance**: Large events with many blocks
  - *Mitigation*: Pagination and lazy loading
- **Data Consistency**: Maintaining order integrity
  - *Mitigation*: Constraints and validation

### 18.2 User Experience Risks
- **Complexity**: Too many options for users
  - *Mitigation*: Progressive disclosure and defaults
- **Learning Curve**: Understanding block concept
  - *Mitigation*: Tutorials and templates
- **Migration**: Existing events without blocks
  - *Mitigation*: Optional feature, gradual adoption