package utils

import (
	"sort"
	"time"

	"github.com/denisAlshanov/stPlaner/internal/models"
)

// CalculateNextOccurrences calculates the next occurrences of a show based on its scheduling configuration
func CalculateNextOccurrences(show *models.Show, maxOccurrences int) []time.Time {
	if show.Status != models.ShowStatusActive {
		return []time.Time{}
	}

	switch show.RepeatPattern {
	case models.RepeatNone:
		return calculateSingleOccurrence(show)
	case models.RepeatDaily:
		return calculateDailyOccurrences(show, maxOccurrences)
	case models.RepeatWeekly:
		return calculateWeeklyOccurrences(show, maxOccurrences, 7)
	case models.RepeatBiweekly:
		return calculateWeeklyOccurrences(show, maxOccurrences, 14)
	case models.RepeatMonthly:
		return calculateMonthlyOccurrences(show, maxOccurrences)
	default:
		return []time.Time{}
	}
}

// calculateSingleOccurrence handles non-repeating shows
func calculateSingleOccurrence(show *models.Show) []time.Time {
	now := time.Now()
	showDateTime := combineDateTime(show.FirstEventDate, show.StartTime)
	
	if showDateTime.After(now) {
		return []time.Time{showDateTime}
	}
	return []time.Time{}
}

// calculateDailyOccurrences handles daily recurring shows
func calculateDailyOccurrences(show *models.Show, maxOccurrences int) []time.Time {
	var occurrences []time.Time
	now := time.Now()
	current := show.FirstEventDate
	
	// Find the first future occurrence
	for combineDateTime(current, show.StartTime).Before(now) {
		current = current.AddDate(0, 0, 1)
	}
	
	// Generate occurrences
	for i := 0; i < maxOccurrences; i++ {
		occurrences = append(occurrences, combineDateTime(current, show.StartTime))
		current = current.AddDate(0, 0, 1)
	}
	
	return occurrences
}

// calculateWeeklyOccurrences handles weekly and biweekly recurring shows
func calculateWeeklyOccurrences(show *models.Show, maxOccurrences int, intervalDays int) []time.Time {
	var occurrences []time.Time
	now := time.Now()
	
	// Get weekdays from scheduling config, default to first event date weekday
	weekdays := []int{int(show.FirstEventDate.Weekday())}
	if show.SchedulingConfig != nil && len(show.SchedulingConfig.Weekdays) > 0 {
		weekdays = show.SchedulingConfig.Weekdays
	}
	
	// Convert int weekdays to time.Weekday for easier handling
	targetWeekdays := make([]time.Weekday, len(weekdays))
	for i, wd := range weekdays {
		targetWeekdays[i] = time.Weekday(wd)
	}
	
	// Start from the first event date
	startDate := show.FirstEventDate
	
	// Generate occurrences for each weekday pattern
	for _, weekday := range targetWeekdays {
		current := findNextWeekday(startDate, weekday)
		
		// Generate occurrences for this weekday
		occurrenceCount := 0
		for occurrenceCount < maxOccurrences/len(targetWeekdays)+1 {
			showDateTime := combineDateTime(current, show.StartTime)
			if showDateTime.After(now) {
				occurrences = append(occurrences, showDateTime)
				occurrenceCount++
			}
			current = current.AddDate(0, 0, intervalDays)
		}
	}
	
	// Sort occurrences by date
	sort.Slice(occurrences, func(i, j int) bool {
		return occurrences[i].Before(occurrences[j])
	})
	
	// Return only the requested number of occurrences
	if len(occurrences) > maxOccurrences {
		occurrences = occurrences[:maxOccurrences]
	}
	
	return occurrences
}

// calculateMonthlyOccurrences handles monthly recurring shows
func calculateMonthlyOccurrences(show *models.Show, maxOccurrences int) []time.Time {
	if show.SchedulingConfig == nil {
		// Default to same day of month as first event
		return calculateMonthlyDayOccurrences(show, maxOccurrences, show.FirstEventDate.Day(), "last_day")
	}
	
	// Check if it's weekday-based monthly scheduling
	if show.SchedulingConfig.MonthlyWeekday != nil && show.SchedulingConfig.MonthlyWeekNumber != nil {
		return calculateMonthlyWeekdayOccurrences(show, maxOccurrences, 
			time.Weekday(*show.SchedulingConfig.MonthlyWeekday), *show.SchedulingConfig.MonthlyWeekNumber)
	}
	
	// Check if it's calendar day-based monthly scheduling
	if show.SchedulingConfig.MonthlyDay != nil {
		fallback := "last_day"
		if show.SchedulingConfig.MonthlyDayFallback != nil {
			fallback = *show.SchedulingConfig.MonthlyDayFallback
		}
		return calculateMonthlyDayOccurrences(show, maxOccurrences, *show.SchedulingConfig.MonthlyDay, fallback)
	}
	
	// Default fallback
	return calculateMonthlyDayOccurrences(show, maxOccurrences, show.FirstEventDate.Day(), "last_day")
}

// calculateMonthlyWeekdayOccurrences calculates monthly occurrences based on weekday and week number
func calculateMonthlyWeekdayOccurrences(show *models.Show, maxOccurrences int, weekday time.Weekday, weekNumber int) []time.Time {
	var occurrences []time.Time
	now := time.Now()
	
	// Start from the month of first event date or current month if past
	current := show.FirstEventDate
	if combineDateTime(current, show.StartTime).Before(now) {
		current = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}
	
	for i := 0; i < maxOccurrences*2; i++ { // Generate more to account for skipped months
		monthStart := time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, current.Location())
		
		var targetDate time.Time
		if weekNumber == -1 {
			// Last occurrence of weekday in month
			targetDate = findLastWeekdayInMonth(monthStart, weekday)
		} else {
			// Nth occurrence of weekday in month
			targetDate = findNthWeekdayInMonth(monthStart, weekday, weekNumber)
		}
		
		if !targetDate.IsZero() {
			// Adjust time to match show start time
			showDateTime := combineDateTime(targetDate, show.StartTime)
			
			if showDateTime.After(now) {
				occurrences = append(occurrences, showDateTime)
				if len(occurrences) >= maxOccurrences {
					break
				}
			}
		}
		
		current = current.AddDate(0, 1, 0) // Next month
	}
	
	return occurrences
}

// calculateMonthlyDayOccurrences calculates monthly occurrences based on calendar day
func calculateMonthlyDayOccurrences(show *models.Show, maxOccurrences int, day int, fallback string) []time.Time {
	var occurrences []time.Time
	now := time.Now()
	
	// Start from the month of first event date or current month if past
	current := show.FirstEventDate
	if combineDateTime(current, show.StartTime).Before(now) {
		current = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}
	
	for i := 0; i < maxOccurrences*2; i++ { // Generate more to account for skipped months
		year, month := current.Year(), current.Month()
		
		// Check if the day exists in this month
		daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, current.Location()).Day()
		
		var targetDay int
		var skip bool
		
		if day <= daysInMonth {
			targetDay = day
		} else {
			// Apply fallback strategy
			switch fallback {
			case "last_day":
				targetDay = daysInMonth
			case "skip":
				skip = true
			default:
				targetDay = daysInMonth
			}
		}
		
		if !skip {
			targetDate := time.Date(year, month, targetDay, 0, 0, 0, 0, current.Location())
			showDateTime := combineDateTime(targetDate, show.StartTime)
			
			if showDateTime.After(now) {
				occurrences = append(occurrences, showDateTime)
				if len(occurrences) >= maxOccurrences {
					break
				}
			}
		}
		
		current = current.AddDate(0, 1, 0) // Next month
	}
	
	return occurrences
}

// Helper functions

// combineDateTime combines a date and time into a single datetime
func combineDateTime(date time.Time, timeOfDay time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(),
		timeOfDay.Hour(), timeOfDay.Minute(), timeOfDay.Second(), 0, date.Location())
}

// findNextWeekday finds the next occurrence of a specific weekday from a given date
func findNextWeekday(from time.Time, targetWeekday time.Weekday) time.Time {
	current := from
	for current.Weekday() != targetWeekday {
		current = current.AddDate(0, 0, 1)
	}
	return current
}

// findNthWeekdayInMonth finds the Nth occurrence of a weekday in a month
func findNthWeekdayInMonth(monthStart time.Time, weekday time.Weekday, n int) time.Time {
	if n < 1 || n > 5 {
		return time.Time{} // Invalid week number
	}
	
	// Find the first occurrence of the weekday in the month
	current := monthStart
	for current.Weekday() != weekday {
		current = current.AddDate(0, 0, 1)
		if current.Month() != monthStart.Month() {
			return time.Time{} // Weekday doesn't exist in this month
		}
	}
	
	// Move to the Nth occurrence
	for i := 1; i < n; i++ {
		current = current.AddDate(0, 0, 7)
		if current.Month() != monthStart.Month() {
			return time.Time{} // Nth occurrence doesn't exist in this month
		}
	}
	
	return current
}

// findLastWeekdayInMonth finds the last occurrence of a weekday in a month
func findLastWeekdayInMonth(monthStart time.Time, weekday time.Weekday) time.Time {
	// Start from the last day of the month
	nextMonth := monthStart.AddDate(0, 1, 0)
	lastDay := nextMonth.AddDate(0, 0, -1)
	
	// Move backwards to find the last occurrence of the weekday
	current := lastDay
	for current.Weekday() != weekday {
		current = current.AddDate(0, 0, -1)
		if current.Month() != monthStart.Month() {
			return time.Time{} // Weekday doesn't exist in this month
		}
	}
	
	return current
}

// GetNextOccurrence returns the single next occurrence of a show
func GetNextOccurrence(show *models.Show) *time.Time {
	occurrences := CalculateNextOccurrences(show, 1)
	if len(occurrences) > 0 {
		return &occurrences[0]
	}
	return nil
}

// ValidateSchedulingConfig validates the scheduling configuration against the repeat pattern
func ValidateSchedulingConfig(pattern models.RepeatPattern, config *models.SchedulingConfig) error {
	if config == nil {
		return nil // Optional for backward compatibility
	}
	
	switch pattern {
	case models.RepeatWeekly, models.RepeatBiweekly:
		if len(config.Weekdays) == 0 {
			return NewValidationError("weekdays required for weekly/biweekly patterns", map[string]interface{}{
				"pattern": pattern,
				"field":   "weekdays",
			})
		}
		
		// Validate weekday values (0-6)
		for _, wd := range config.Weekdays {
			if wd < 0 || wd > 6 {
				return NewValidationError("invalid weekday value", map[string]interface{}{
					"weekday": wd,
					"valid_range": "0-6 (0=Sunday, 6=Saturday)",
				})
			}
		}
		
		if config.MonthlyWeekday != nil || config.MonthlyDay != nil {
			return NewValidationError("monthly fields not allowed for weekly/biweekly patterns", map[string]interface{}{
				"pattern": pattern,
			})
		}
		
	case models.RepeatMonthly:
		hasWeekdayConfig := config.MonthlyWeekday != nil && config.MonthlyWeekNumber != nil
		hasDayConfig := config.MonthlyDay != nil
		
		if !hasWeekdayConfig && !hasDayConfig {
			return NewValidationError("either weekday-based or day-based config required for monthly pattern", map[string]interface{}{
				"pattern": pattern,
			})
		}
		if hasWeekdayConfig && hasDayConfig {
			return NewValidationError("cannot specify both weekday-based and day-based config", map[string]interface{}{
				"pattern": pattern,
			})
		}
		if len(config.Weekdays) > 0 {
			return NewValidationError("weekdays not allowed for monthly patterns", map[string]interface{}{
				"pattern": pattern,
			})
		}
		
		// Validate monthly weekday config
		if hasWeekdayConfig {
			if *config.MonthlyWeekday < 0 || *config.MonthlyWeekday > 6 {
				return NewValidationError("invalid monthly weekday value", map[string]interface{}{
					"weekday": *config.MonthlyWeekday,
					"valid_range": "0-6 (0=Sunday, 6=Saturday)",
				})
			}
			if *config.MonthlyWeekNumber < -1 || *config.MonthlyWeekNumber == 0 || *config.MonthlyWeekNumber > 4 {
				return NewValidationError("invalid monthly week number", map[string]interface{}{
					"week_number": *config.MonthlyWeekNumber,
					"valid_values": "1, 2, 3, 4, or -1 (for last)",
				})
			}
		}
		
		// Validate monthly day config
		if hasDayConfig {
			if *config.MonthlyDay < 1 || *config.MonthlyDay > 31 {
				return NewValidationError("invalid monthly day value", map[string]interface{}{
					"day": *config.MonthlyDay,
					"valid_range": "1-31",
				})
			}
			if config.MonthlyDayFallback != nil {
				fallback := *config.MonthlyDayFallback
				if fallback != "last_day" && fallback != "skip" {
					return NewValidationError("invalid monthly day fallback", map[string]interface{}{
						"fallback": fallback,
						"valid_values": []string{"last_day", "skip"},
					})
				}
			}
		}
	}
	
	return nil
}

// Event Generation Functions

// GenerateEventsForShow generates concrete events from a show template for the next 3 months
func GenerateEventsForShow(show *models.Show, generateUntil time.Time) ([]models.Event, error) {
	var events []models.Event
	
	if show.Status != models.ShowStatusActive {
		return events, nil
	}
	
	// Get show occurrences using existing scheduling logic
	occurrences := CalculateNextOccurrences(show, 1000) // Large number to cover 3 months
	
	now := time.Now()
	for _, occurrence := range occurrences {
		if occurrence.After(generateUntil) {
			break // Stop after generateUntil date
		}
		
		if occurrence.Before(now) {
			continue // Skip past dates
		}
		
		// Calculate end time
		endTime := occurrence.Add(time.Duration(show.LengthMinutes) * time.Minute)
		
		event := models.Event{
			ShowID:        show.ID,
			UserID:        show.UserID,
			StartDateTime: occurrence,
			EndDateTime:   endTime,
			Status:        models.EventStatusScheduled,
			IsCustomized:  false,
			ShowVersion:   show.Version,
			GeneratedAt:   now,
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// GetThreeMonthHorizon returns the date 3 full months from today
func GetThreeMonthHorizon() time.Time {
	now := time.Now()
	// Get end of current month
	endOfCurrentMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)
	// Add 3 more months
	return endOfCurrentMonth.AddDate(0, 3, 0)
}

// CalculateEventEndTime calculates the end time for an event
func CalculateEventEndTime(event *models.Event, show *models.Show) time.Time {
	duration := show.LengthMinutes
	if event.LengthMinutes != nil {
		duration = *event.LengthMinutes
	}
	return event.StartDateTime.Add(time.Duration(duration) * time.Minute)
}

// GetEffectiveEventData returns the effective event data combining event overrides with show defaults
func GetEffectiveEventData(event *models.Event, show *models.Show) EventData {
	return EventData{
		Title:          coalesce(event.EventTitle, &show.ShowName),
		Description:    coalesce(event.EventDescription, getShowDescription(show)),
		YouTubeKey:     coalesce(event.YouTubeKey, &show.YouTubeKey),
		AdditionalKey:  coalescePtr(event.AdditionalKey, show.AdditionalKey),
		ZoomMeetingURL: coalescePtr(event.ZoomMeetingURL, show.ZoomMeetingURL),
		ZoomMeetingID:  coalescePtr(event.ZoomMeetingID, show.ZoomMeetingID),
		ZoomPasscode:   coalescePtr(event.ZoomPasscode, show.ZoomPasscode),
		Duration:       coalesceInt(event.LengthMinutes, &show.LengthMinutes),
		StartTime:      event.StartDateTime,
		EndTime:        event.EndDateTime,
	}
}

// EventData represents the effective data for an event
type EventData struct {
	Title          string
	Description    string
	YouTubeKey     string
	AdditionalKey  *string
	ZoomMeetingURL *string
	ZoomMeetingID  *string
	ZoomPasscode   *string
	Duration       int
	StartTime      time.Time
	EndTime        time.Time
}

// Helper functions for coalescing values

// coalesce returns the first non-nil string value
func coalesce(values ...*string) string {
	for _, v := range values {
		if v != nil && *v != "" {
			return *v
		}
	}
	return ""
}

// coalesceInt returns the first non-nil int value
func coalesceInt(values ...*int) int {
	for _, v := range values {
		if v != nil {
			return *v
		}
	}
	return 0
}

// coalescePtr returns the first non-nil string pointer value
func coalescePtr(values ...*string) *string {
	for _, v := range values {
		if v != nil && *v != "" {
			return v
		}
	}
	return nil
}

// getShowDescription extracts a description from show metadata or returns empty string
func getShowDescription(show *models.Show) *string {
	if show.Metadata != nil {
		if desc, ok := show.Metadata["description"].(string); ok && desc != "" {
			return &desc
		}
	}
	return nil
}

// FilterCustomizedEvents filters events that have been customized by users
func FilterCustomizedEvents(events []models.Event) []models.Event {
	var customized []models.Event
	for _, event := range events {
		if event.IsCustomized {
			customized = append(customized, event)
		}
	}
	return customized
}

// FilterNonCustomizedEvents filters events that have not been customized by users
func FilterNonCustomizedEvents(events []models.Event) []models.Event {
	var nonCustomized []models.Event
	for _, event := range events {
		if !event.IsCustomized {
			nonCustomized = append(nonCustomized, event)
		}
	}
	return nonCustomized
}

// ValidateEventTiming validates that event timing is valid
func ValidateEventTiming(event *models.Event, show *models.Show) error {
	// Event must be in the future (for new events)
	if event.StartDateTime.Before(time.Now()) && event.Status == models.EventStatusScheduled {
		return NewValidationError("cannot schedule events in the past", map[string]interface{}{
			"start_datetime": event.StartDateTime,
			"current_time":   time.Now(),
		})
	}
	
	// Event must belong to user
	if event.UserID != show.UserID {
		return NewValidationError("event user must match show user", map[string]interface{}{
			"event_user_id": event.UserID,
			"show_user_id":  show.UserID,
		})
	}
	
	// Duration must be reasonable
	duration := event.EndDateTime.Sub(event.StartDateTime)
	if duration <= 0 || duration > 24*time.Hour {
		return NewValidationError("event duration must be between 1 minute and 24 hours", map[string]interface{}{
			"duration_minutes": duration.Minutes(),
			"start_datetime":   event.StartDateTime,
			"end_datetime":     event.EndDateTime,
		})
	}
	
	return nil
}