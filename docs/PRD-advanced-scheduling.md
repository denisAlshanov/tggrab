# Product Requirements Document: Advanced Show Scheduling Patterns

## 1. Executive Summary

### 1.1 Purpose
This PRD defines the requirements for enhancing the show planning functionality with advanced recurring patterns, enabling more sophisticated scheduling options for weekly, biweekly, and monthly shows.

### 1.2 Scope
The enhancement will allow users to:
- Specify specific days of the week for weekly and biweekly shows
- Configure monthly shows to repeat on specific weekdays (e.g., "first Monday", "last Friday")
- Configure monthly shows to repeat on specific calendar days (e.g., "15th of every month")
- Maintain backward compatibility with existing simple scheduling

### 1.3 Background
The current scheduling system only supports basic recurring patterns without granular control over which days shows occur. This limitation prevents content creators from setting up realistic streaming schedules that align with their audience's availability and their own preferred streaming days.

## 2. Current State vs. Desired State

### 2.1 Current Limitations
- **Weekly/Biweekly Shows**: Only repeat on the same day of the week as the first event
- **Monthly Shows**: Only repeat on the same calendar date each month
- **No Flexibility**: Cannot specify multiple days or different scheduling patterns

### 2.2 Desired Enhancements
- **Weekly/Biweekly Shows**: Support for multiple specific weekdays (e.g., Monday, Wednesday, Friday)
- **Monthly Shows**: Support for weekday-based scheduling (e.g., "second Tuesday of every month")
- **Monthly Shows**: Support for calendar day-based scheduling with fallback logic
- **Backward Compatibility**: Existing shows continue to work without modification

## 3. User Stories

### 3.1 Weekly/Biweekly Scheduling
1. **As a content creator**, I want to schedule a weekly show on Mondays and Wednesdays so that I can maintain a consistent twice-weekly streaming schedule.

2. **As a content creator**, I want to schedule a biweekly show on specific weekdays so that I can alternate between different content themes.

3. **As a content creator**, I want to see upcoming show dates that account for my selected weekdays so that I can plan my content accordingly.

### 3.2 Monthly Scheduling
4. **As a content creator**, I want to schedule a monthly show on the "first Monday of every month" so that I can have a predictable monthly schedule.

5. **As a content creator**, I want to schedule a monthly show on the "last Friday of every month" for month-end wrap-ups.

6. **As a content creator**, I want to schedule a monthly show on the 15th of every month, with smart handling for months that don't have that date.

## 4. Functional Requirements

### 4.1 Enhanced Data Model

#### 4.1.1 Scheduling Configuration
```go
type SchedulingConfig struct {
    // For weekly and biweekly patterns
    Weekdays []time.Weekday `json:"weekdays,omitempty"`
    
    // For monthly patterns - weekday-based
    MonthlyWeekday     *time.Weekday `json:"monthly_weekday,omitempty"`     // Monday, Tuesday, etc.
    MonthlyWeekNumber  *int          `json:"monthly_week_number,omitempty"` // 1, 2, 3, 4, or -1 for last
    
    // For monthly patterns - calendar day-based
    MonthlyDay         *int          `json:"monthly_day,omitempty"`         // 1-31
    MonthlyDayFallback *string       `json:"monthly_day_fallback,omitempty"` // "last_day", "skip"
}

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
```

#### 4.1.2 Updated Show Model
```go
type Show struct {
    // ... existing fields ...
    
    // Enhanced scheduling configuration
    SchedulingConfig *SchedulingConfig `json:"scheduling_config,omitempty" db:"scheduling_config"`
}
```

### 4.2 Database Schema Changes

#### 4.2.1 New Database Fields
```sql
-- Add scheduling configuration as JSONB
ALTER TABLE shows ADD COLUMN scheduling_config JSONB;

-- Create index for scheduling queries
CREATE INDEX idx_shows_scheduling_config ON shows USING GIN (scheduling_config);
```

#### 4.2.2 Scheduling Config Structure
```json
{
  "weekdays": [1, 3, 5],                    // Monday, Wednesday, Friday (for weekly/biweekly)
  "monthly_weekday": 1,                     // Monday (for monthly weekday-based)
  "monthly_week_number": 2,                 // Second week (1, 2, 3, 4, or -1 for last)
  "monthly_day": 15,                        // 15th of month (for monthly day-based)
  "monthly_day_fallback": "last_day"        // Fallback strategy
}
```

### 4.3 API Enhancements

#### 4.3.1 Enhanced Create Show Request
```json
{
  "show_name": "Weekly Tech Talk",
  "youtube_key": "stream-key",
  "start_time": "14:00:00",
  "length_minutes": 60,
  "first_event_date": "2025-01-15",
  "repeat_pattern": "weekly",
  "scheduling_config": {
    "weekdays": [1, 3, 5]  // Monday, Wednesday, Friday
  }
}
```

#### 4.3.2 Monthly Weekday-Based Example
```json
{
  "show_name": "Monthly Management Meeting",
  "repeat_pattern": "monthly",
  "scheduling_config": {
    "monthly_weekday": 1,      // Monday
    "monthly_week_number": 1   // First Monday of every month
  }
}
```

#### 4.3.3 Monthly Calendar Day-Based Example
```json
{
  "show_name": "Mid-Month Review",
  "repeat_pattern": "monthly",
  "scheduling_config": {
    "monthly_day": 15,
    "monthly_day_fallback": "last_day"  // Use last day of month if 15th doesn't exist
  }
}
```

### 4.4 Validation Rules

#### 4.4.1 Weekly/Biweekly Patterns
- `weekdays` array must contain 1-7 unique values
- Each weekday must be valid (0-6, where 0=Sunday, 6=Saturday)
- If not specified, defaults to the weekday of `first_event_date`

#### 4.4.2 Monthly Weekday-Based Patterns
- `monthly_weekday` must be valid (0-6)
- `monthly_week_number` must be 1, 2, 3, 4, or -1
- Both fields must be specified together
- Cannot be combined with `monthly_day`

#### 4.4.3 Monthly Calendar Day-Based Patterns
- `monthly_day` must be 1-31
- `monthly_day_fallback` must be valid enum value
- Cannot be combined with monthly weekday fields

### 4.5 Scheduling Logic

#### 4.5.1 Weekly/Biweekly Calculation
```go
func calculateWeeklyOccurrences(firstDate time.Time, weekdays []time.Weekday, isWeekly bool) []time.Time {
    var occurrences []time.Time
    interval := 7 // days
    if !isWeekly {
        interval = 14 // biweekly
    }
    
    // Find all matching weekdays in the pattern
    for _, weekday := range weekdays {
        current := findNextWeekday(firstDate, weekday)
        // Generate occurrences for this weekday
        for i := 0; i < maxOccurrences; i++ {
            occurrences = append(occurrences, current)
            current = current.AddDate(0, 0, interval)
        }
    }
    
    // Sort by date
    sort.Slice(occurrences, func(i, j int) bool {
        return occurrences[i].Before(occurrences[j])
    })
    
    return occurrences
}
```

#### 4.5.2 Monthly Weekday Calculation
```go
func calculateMonthlyWeekdayOccurrences(firstDate time.Time, weekday time.Weekday, weekNumber int) []time.Time {
    var occurrences []time.Time
    current := firstDate
    
    for i := 0; i < maxOccurrences; i++ {
        // Find the specified weekday in the specified week of the month
        monthStart := time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, current.Location())
        
        var targetDate time.Time
        if weekNumber == -1 {
            // Last occurrence of weekday in month
            targetDate = findLastWeekdayInMonth(monthStart, weekday)
        } else {
            // Nth occurrence of weekday in month
            targetDate = findNthWeekdayInMonth(monthStart, weekday, weekNumber)
        }
        
        // Adjust time to match show start time
        targetDate = time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(),
            firstDate.Hour(), firstDate.Minute(), firstDate.Second(), 0, firstDate.Location())
        
        occurrences = append(occurrences, targetDate)
        current = current.AddDate(0, 1, 0) // Next month
    }
    
    return occurrences
}
```

#### 4.5.3 Monthly Calendar Day Calculation
```go
func calculateMonthlyDayOccurrences(firstDate time.Time, day int, fallback MonthlyDayFallback) []time.Time {
    var occurrences []time.Time
    current := firstDate
    
    for i := 0; i < maxOccurrences; i++ {
        year, month := current.Year(), current.Month()
        
        // Check if the day exists in this month
        daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, current.Location()).Day()
        
        var targetDay int
        if day <= daysInMonth {
            targetDay = day
        } else {
            // Apply fallback strategy
            switch fallback {
            case MonthlyDayFallbackLastDay:
                targetDay = daysInMonth
            case MonthlyDayFallbackSkip:
                current = current.AddDate(0, 1, 0)
                continue
            }
        }
        
        targetDate := time.Date(year, month, targetDay,
            firstDate.Hour(), firstDate.Minute(), firstDate.Second(), 0, firstDate.Location())
        
        occurrences = append(occurrences, targetDate)
        current = current.AddDate(0, 1, 0)
    }
    
    return occurrences
}
```

## 5. API Changes

### 5.1 Request/Response Updates

#### 5.1.1 Enhanced Create Show Request
```go
type CreateShowRequest struct {
    // ... existing fields ...
    SchedulingConfig *SchedulingConfig `json:"scheduling_config,omitempty"`
}

type SchedulingConfig struct {
    Weekdays           []int    `json:"weekdays,omitempty" binding:"dive,min=0,max=6"`
    MonthlyWeekday     *int     `json:"monthly_weekday,omitempty" binding:"omitempty,min=0,max=6"`
    MonthlyWeekNumber  *int     `json:"monthly_week_number,omitempty" binding:"omitempty,oneof=1 2 3 4 -1"`
    MonthlyDay         *int     `json:"monthly_day,omitempty" binding:"omitempty,min=1,max=31"`
    MonthlyDayFallback *string  `json:"monthly_day_fallback,omitempty" binding:"omitempty,oneof=last_day skip"`
}
```

#### 5.1.2 Enhanced Show List Response
```go
type ShowListItem struct {
    // ... existing fields ...
    SchedulingConfig   *SchedulingConfig `json:"scheduling_config,omitempty"`
    NextOccurrences    []time.Time       `json:"next_occurrences,omitempty"` // Next 3 occurrences
}
```

### 5.2 Validation Logic

#### 5.2.1 Cross-Field Validation
```go
func validateSchedulingConfig(pattern RepeatPattern, config *SchedulingConfig) error {
    if config == nil {
        return nil // Optional for backward compatibility
    }
    
    switch pattern {
    case RepeatWeekly, RepeatBiweekly:
        if len(config.Weekdays) == 0 {
            return errors.New("weekdays required for weekly/biweekly patterns")
        }
        if config.MonthlyWeekday != nil || config.MonthlyDay != nil {
            return errors.New("monthly fields not allowed for weekly/biweekly patterns")
        }
        
    case RepeatMonthly:
        hasWeekdayConfig := config.MonthlyWeekday != nil && config.MonthlyWeekNumber != nil
        hasDayConfig := config.MonthlyDay != nil
        
        if !hasWeekdayConfig && !hasDayConfig {
            return errors.New("either weekday-based or day-based config required for monthly pattern")
        }
        if hasWeekdayConfig && hasDayConfig {
            return errors.New("cannot specify both weekday-based and day-based config")
        }
        if len(config.Weekdays) > 0 {
            return errors.New("weekdays not allowed for monthly patterns")
        }
    }
    
    return nil
}
```

## 6. Migration Strategy

### 6.1 Database Migration
```sql
-- Migration Version 6: Add advanced scheduling configuration
DO $$ 
BEGIN
    -- Add scheduling_config column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'shows' AND column_name = 'scheduling_config') THEN
        ALTER TABLE shows ADD COLUMN scheduling_config JSONB;
        
        -- Create index for scheduling queries
        CREATE INDEX idx_shows_scheduling_config ON shows USING GIN (scheduling_config);
        
        -- Migrate existing shows to new format
        UPDATE shows SET scheduling_config = jsonb_build_object(
            'weekdays', ARRAY[EXTRACT(DOW FROM first_event_date)::int]
        ) WHERE repeat_pattern IN ('weekly', 'biweekly') AND scheduling_config IS NULL;
        
        -- For monthly shows, use the day of month approach
        UPDATE shows SET scheduling_config = jsonb_build_object(
            'monthly_day', EXTRACT(DAY FROM first_event_date)::int,
            'monthly_day_fallback', 'last_day'
        ) WHERE repeat_pattern = 'monthly' AND scheduling_config IS NULL;
    END IF;
END $$;
```

### 6.2 Backward Compatibility
- Existing shows without `scheduling_config` will use default behavior
- API continues to accept requests without scheduling config
- Default scheduling config is generated based on `first_event_date`

## 7. Examples and Use Cases

### 7.1 Weekly Show Examples

#### 7.1.1 Twice Weekly Show
```json
{
  "show_name": "Code Review Sessions",
  "repeat_pattern": "weekly",
  "scheduling_config": {
    "weekdays": [2, 4]  // Tuesday and Thursday
  }
}
```

#### 7.1.2 Weekend Show
```json
{
  "show_name": "Weekend Gaming",
  "repeat_pattern": "weekly",
  "scheduling_config": {
    "weekdays": [6, 0]  // Saturday and Sunday
  }
}
```

### 7.2 Monthly Show Examples

#### 7.2.1 First Monday Monthly
```json
{
  "show_name": "Monthly Planning",
  "repeat_pattern": "monthly",
  "scheduling_config": {
    "monthly_weekday": 1,      // Monday
    "monthly_week_number": 1   // First week
  }
}
```

#### 7.2.2 Last Friday Monthly
```json
{
  "show_name": "Month-End Wrap-up",
  "repeat_pattern": "monthly",
  "scheduling_config": {
    "monthly_weekday": 5,      // Friday
    "monthly_week_number": -1  // Last occurrence
  }
}
```

#### 7.2.3 15th of Every Month
```json
{
  "show_name": "Mid-Month Check-in",
  "repeat_pattern": "monthly",
  "scheduling_config": {
    "monthly_day": 15,
    "monthly_day_fallback": "last_day"  // Use last day if 15th doesn't exist
  }
}
```

## 8. Error Handling

### 8.1 Validation Errors
- Invalid weekday combinations
- Conflicting monthly configuration
- Missing required fields for pattern type

### 8.2 Scheduling Errors
- No valid dates found for pattern
- Infinite loop prevention in date calculation
- Timezone edge cases

## 9. Testing Requirements

### 9.1 Unit Tests
- Weekday calculation logic
- Monthly weekday calculation (including edge cases like 5th occurrence)
- Monthly calendar day calculation with fallbacks
- Validation logic for all pattern types

### 9.2 Integration Tests
- API endpoints with new scheduling config
- Database migration testing
- Backward compatibility verification

### 9.3 Edge Cases
- Months with different numbers of days
- Leap years
- DST transitions
- Time zone changes
- 5th occurrence of weekday (some months don't have it)

## 10. Performance Considerations

### 10.1 Database Optimization
- JSONB indexing for scheduling config queries
- Efficient date calculation algorithms
- Limit on maximum occurrences calculated

### 10.2 Caching Strategy
- Cache calculated occurrences for frequently accessed shows
- Invalidate cache when scheduling config changes

## 11. Future Enhancements

### 11.1 Phase 2 Features
- Custom recurring patterns (e.g., "every 3 weeks")
- Exclusion dates (skip specific dates)
- Seasonal scheduling (different patterns for different seasons)
- Holiday awareness (skip or reschedule for holidays)

### 11.2 Advanced Features
- Multiple time slots per day
- Different scheduling per show segment
- Audience timezone optimization
- AI-powered optimal scheduling suggestions

## 12. Success Metrics

### 12.1 Technical Metrics
- Migration success rate (100% of existing shows migrated)
- API response time (< 200ms for scheduling calculations)
- Accuracy of date calculations (100% for test cases)

### 12.2 User Metrics
- Adoption rate of advanced scheduling features
- Reduction in show creation errors
- User satisfaction with scheduling flexibility

## 13. Implementation Timeline

### 13.1 Phase 1 (Week 1-2)
- Data model updates
- Database migration
- Core scheduling logic

### 13.2 Phase 2 (Week 3-4)
- API enhancements
- Validation logic
- Error handling

### 13.3 Phase 3 (Week 5-6)
- Testing and bug fixes
- Documentation updates
- Performance optimization

## 14. Risks and Mitigation

### 14.1 Technical Risks
- **Complex Date Calculations**: Mitigate with comprehensive testing and proven algorithms
- **Migration Issues**: Mitigate with rollback plan and staged deployment
- **Performance Impact**: Mitigate with caching and query optimization

### 14.2 User Experience Risks
- **Complexity Overwhelm**: Mitigate with clear documentation and optional features
- **Breaking Changes**: Mitigate with backward compatibility and gradual rollout