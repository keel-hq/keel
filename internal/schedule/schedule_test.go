package schedule

import (
	"testing"
	"time"

	"github.com/rusenask/cron"
	"github.com/stretchr/testify/assert"
)

func mustParseTime(layout, value string) time.Time {
	t, _ := time.Parse(layout, value)
	return t
}

func TestFindPrevScheduledTime(t *testing.T) {

	tests := []struct {
		name         string
		cronSchedule string
		now          time.Time
		expected     string // Expected previous time in RFC3339 format
	}{
		{
			name:         "hourly schedule",
			cronSchedule: "0 0 * * * *", // Every hour at minute 0
			now:          mustParseTime(time.RFC3339, "2023-05-15T14:30:00Z"),
			expected:     "2023-05-15T14:00:00Z", // Same day at 14:00
		},
		{
			name:         "every 15 minutes",
			cronSchedule: "0 */15 * * * *", // Every 15 minutes
			now:          mustParseTime(time.RFC3339, "2023-05-15T14:30:25Z"),
			expected:     "2023-05-15T14:30:00Z", // Same day at 14:15
		},
		{
			name:         "every 15 minutes II",
			cronSchedule: "0 */15 * * * *", // Every 15 minutes
			now:          mustParseTime(time.RFC3339, "2023-05-15T14:29:00Z"),
			expected:     "2023-05-15T14:15:00Z", // Same day at 14:15
		},
		{
			name:         "daily schedule",
			cronSchedule: "0 0 0 * * *", // Midnight every day
			now:          mustParseTime(time.RFC3339, "2023-05-15T14:30:00Z"),
			expected:     "2023-05-15T00:00:00Z", // Same day at midnight
		},
		{
			name:         "weekly schedule",
			cronSchedule: "0 0 0 * * 0", // Midnight every Sunday
			now:          mustParseTime(time.RFC3339, "2023-05-15T14:30:00Z"),
			expected:     "2023-05-14T00:00:00Z", // Previous Sunday
		},
		{
			name:         "monthly schedule",
			cronSchedule: "0 0 0 1 * *", // 1st day of month
			now:          mustParseTime(time.RFC3339, "2023-05-15T14:30:00Z"),
			expected:     "", // 1st of May
		},
		{
			name:         "specific weekday",
			cronSchedule: "0 0 12 * * 1", // Monday at noon
			now:          mustParseTime(time.RFC3339, "2023-05-15T14:30:00Z"),
			expected:     "2023-05-15T12:00:00Z", // Same Monday at noon (this is in the future from now)
		},
		{
			name:         "specific time of day",
			cronSchedule: "0 30 9 * * *", // 9:30 every day
			now:          mustParseTime(time.RFC3339, "2023-05-15T14:30:00Z"),
			expected:     "2023-05-15T09:30:00Z", // Same day at 9:30
		},
		{
			name:         "schedule with seconds",
			cronSchedule: "0 15 */10 * * *", // 15 seconds past every 10 minutes
			now:          mustParseTime(time.RFC3339, "2023-05-15T14:30:00Z"),
			expected:     "2023-05-15T10:15:00Z",
		},
		{
			name:         "no previous occurrences",
			cronSchedule: "0 0 0 31 2 *", // February 31st (impossible date)
			now:          mustParseTime(time.RFC3339, "2023-05-15T14:30:00Z"),
			expected:     "", // Should return zero time
		},
		{
			name:         "exactly at scheduled time",
			cronSchedule: "0 30 14 15 5 *", // 14:30 on May 15
			now:          mustParseTime(time.RFC3339, "2023-05-15T14:30:00Z"),
			expected:     "2023-05-15T14:30:00Z", // Exactly now
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := cron.Parse(tt.cronSchedule)
			if err != nil && tt.expected != "" {
				t.Fatalf("Failed to parse cron schedule: %v", err)
			}

			var expectedTime time.Time
			if tt.expected != "" {
				expectedTime, _ = time.Parse(time.RFC3339, tt.expected)
			}

			result, _ := findPrevScheduledTime(schedule, tt.now)

			if tt.expected == "" {
				assert.True(t, result.IsZero(), "Expected zero time for %s", tt.name)
			} else {
				assert.Equal(t, expectedTime, result, "Incorrect previous time for %s", tt.name)
			}
		})
	}
}

func TestFindPrevScheduledTime_ActualImplementation(t *testing.T) {
	// This test runs against the real system clock and cron implementation
	tests := []struct {
		name         string
		cronSchedule string
		checkFunc    func(time.Time) bool
	}{
		{
			name:         "hourly schedule within past day",
			cronSchedule: "0 0 * * * *", // Every hour at minute 0
			checkFunc: func(result time.Time) bool {
				now := time.Now()
				// Should be within the past day
				return result.Before(now) &&
					now.Sub(result) < 24*time.Hour &&
					result.Minute() == 0 &&
					result.Second() == 0
			},
		},
		{
			name:         "daily schedule within past week",
			cronSchedule: "0 0 0 * * *", // Midnight every day
			checkFunc: func(result time.Time) bool {
				now := time.Now()
				// Should be within the past week, at midnight
				return result.Before(now) &&
					now.Sub(result) < 7*24*time.Hour &&
					result.Hour() == 0 &&
					result.Minute() == 0 &&
					result.Second() == 0
			},
		},
		{
			name:         "recent 15-minute mark",
			cronSchedule: "0 */15 * * * *", // Every 15 minutes
			checkFunc: func(result time.Time) bool {
				now := time.Now()
				// Should be within the past hour, and minute should be 0, 15, 30, or 45
				return result.Before(now) &&
					now.Sub(result) < time.Hour &&
					(result.Minute() == 0 || result.Minute() == 15 ||
						result.Minute() == 30 || result.Minute() == 45) &&
					result.Second() == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := cron.Parse(tt.cronSchedule)
			if err != nil {
				t.Fatalf("Failed to parse cron schedule: %v", err)
			}

			result, _ := findPrevScheduledTime(schedule, time.Now())

			assert.False(t, result.IsZero(), "Should not return zero time for %s", tt.name)
			assert.True(t, tt.checkFunc(result), "Result time didn't meet expectations for %s: %v", tt.name, result)
		})
	}
}

// Test for edge cases and performance
func TestFindPrevScheduledTime_EdgeCases(t *testing.T) {
	// Lookback beyond our 7-day window
	t.Run("schedule beyond lookback window", func(t *testing.T) {
		now := time.Now()
		cronStr := "0 0 1 1 *" // January 1st at midnight

		if now.Month() == time.January && now.Day() <= 8 {
			t.Skip("Skipping test as we're too close to January 1st")
		}

		schedule, _ := cron.Parse(cronStr)
		result, _ := findPrevScheduledTime(schedule, now)

		// If now is not in January or we're past Jan 8th, the previous Jan 1st
		// would be beyond our 7-day lookback
		assert.True(t, result.IsZero(), "Expected zero time for schedule beyond lookback")
	})

	// Performance test for frequently occurring schedule
	t.Run("performance for frequent schedule", func(t *testing.T) {
		cronStr := "* * * * *" // Every minute
		schedule, _ := cron.Parse(cronStr)
		now := time.Now()

		start := time.Now()
		result, _ := findPrevScheduledTime(schedule, now)
		duration := time.Since(start)

		assert.False(t, result.IsZero(), "Should find a result for every minute schedule")
		assert.True(t, duration < 100*time.Millisecond,
			"Finding prev time took too long: %v", duration)
	})
}
