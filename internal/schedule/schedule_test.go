package schedule

import (
	"strings"
	"testing"
	"time"

	"github.com/keel-hq/keel/types"
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

func TestParseUpdateSchedule(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		wantErr     bool
		errMsg      string
		wantNil     bool
	}{
		{
			name:        "nil annotations",
			annotations: nil,
			wantNil:     true,
		},
		{
			name:        "empty annotations",
			annotations: map[string]string{},
			wantNil:     true,
		},
		{
			name: "invalid cron format",
			annotations: map[string]string{
				types.KeelUpdateScheduleCronTabs: "invalid|5m",
			},
			wantErr: true,
			errMsg:  "Expected 5 to 6 fields, found 1: invalid",
		},
		{
			name: "invalid duration format",
			annotations: map[string]string{
				types.KeelUpdateScheduleCronTabs: "0 * * * * *|invalid",
			},
			wantErr: true,
			errMsg:  "time: invalid duration \"invalid\"",
		},
		{
			name: "missing duration part",
			annotations: map[string]string{
				types.KeelUpdateScheduleCronTabs: "0 * * * * *",
			},
			wantErr: true,
			errMsg:  "invalid schedule format '0 * * * * *', expected 'CRONTAB|DURATION,CRONTAB2|DURATION2'",
		},
		{
			name: "valid single schedule",
			annotations: map[string]string{
				types.KeelUpdateScheduleCronTabs: "0 */5 * * * *|10m",
			},
			wantErr: false,
		},
		{
			name: "valid multiple schedules",
			annotations: map[string]string{
				types.KeelUpdateScheduleCronTabs: "0 0 0 * * *|1h, 0 */30 * * * *|15m",
			},
			wantErr: false,
		},
		{
			name: "empty schedule in list",
			annotations: map[string]string{
				types.KeelUpdateScheduleCronTabs: "0 0 0 * * *|1h, , 0 */30 * * * *|15m",
			},
			wantErr: false,
		},
		{
			name: "whitespace in schedule",
			annotations: map[string]string{
				types.KeelUpdateScheduleCronTabs: "  0 0 0 * * *  |  1h  ",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := ParseUpdateSchedule(tt.annotations)
			if tt.wantNil {
				if schedule != nil {
					t.Errorf("ParseUpdateSchedule() = %v, want nil", schedule)
				}
				return
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseUpdateSchedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("ParseUpdateSchedule() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}
			if !tt.wantErr && schedule == nil {
				t.Error("ParseUpdateSchedule() = nil, want non-nil")
			}
		})
	}
}

func TestIsUpdateAllowed(t *testing.T) {
	// Use a fixed time that aligns with our cron schedules
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		schedule   string
		lastUpdate time.Time
		checkTime  time.Time
		want       bool
		wantErr    bool
	}{
		{
			name:     "nil schedule",
			schedule: "",
			want:     true,
		},
		{
			name:       "within window",
			schedule:   "0 0 * * * *|5m", // every hour at minute 0 with 5m window
			lastUpdate: baseTime.Add(-10 * time.Minute),
			checkTime:  baseTime, // 12:00:00, which matches the schedule
			want:       true,
		},
		{
			name:       "outside window",
			schedule:   "0 0 * * * *|5m", // every hour at minute 0 with 5m window
			lastUpdate: baseTime.Add(-10 * time.Minute),
			checkTime:  baseTime.Add(30 * time.Minute), // 12:30, which is outside the window
			want:       false,
		},
		{
			name:       "multiple schedules - one matches",
			schedule:   "0 0 0 * * *|1h, 0 0 * * * *|10m", // daily at midnight with 1h window, and hourly with 10m window
			lastUpdate: baseTime.Add(-30 * time.Minute),
			checkTime:  baseTime, // 12:00:00, which matches the hourly schedule
			want:       true,
		},
		{
			name:       "within cooldown period",
			schedule:   "0 0 * * * *|5m", // every hour with 5m window
			lastUpdate: baseTime.Add(-2 * time.Minute),
			checkTime:  baseTime,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var schedule *UpdateSchedule
			var err error

			if tt.schedule != "" {
				schedule, err = ParseUpdateSchedule(map[string]string{
					types.KeelUpdateScheduleCronTabs: tt.schedule,
				})
				if err != nil {
					t.Fatalf("Failed to parse schedule: %v", err)
				}
			}

			got, err := schedule.IsUpdateAllowed(tt.lastUpdate, tt.checkTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsUpdateAllowed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsUpdateAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return s != "" && substr != "" && s != substr && len(s) > len(substr) && strings.Contains(s, substr)
}
