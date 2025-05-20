package schedule

import (
	"fmt"
	"strings"
	"time"

	"github.com/keel-hq/keel/types"
	"github.com/rusenask/cron"
)

// UpdateSchedule represents the update schedule configuration
type UpdateSchedule struct {
	CronTabs []string
	Duration string
	CoolDown string
}

// ParseUpdateSchedule parses the update schedule from annotations
func ParseUpdateSchedule(annotations map[string]string) (*UpdateSchedule, error) {
	if annotations == nil {
		return nil, nil
	}

	schedule := &UpdateSchedule{
		CronTabs: strings.Split(annotations[types.KeelUpdateScheduleCronTabs], ","),
		Duration: annotations[types.KeelUpdateScheduleDurationMinutes],
		CoolDown: annotations[types.KeelUpdateScheduleCoolDownMinutes],
	}

	// If no schedule is configured, return nil
	if len(schedule.CronTabs) == 0 || schedule.CronTabs[0] == "" {
		return nil, nil
	}

	// Validate cron schedules
	for _, cronStr := range schedule.CronTabs {
		cronStr = strings.TrimSpace(cronStr)
		if cronStr == "" {
			continue
		}
		_, err := cron.Parse(cronStr)
		if err != nil {
			return nil, fmt.Errorf("invalid cron schedule '%s': %v", cronStr, err)
		}
	}

	// Validate duration if provided
	if schedule.Duration != "" {
		_, err := time.ParseDuration(schedule.Duration)
		if err != nil {
			return nil, fmt.Errorf("invalid duration: %v", err)
		}
	}

	// Validate cooldown if provided
	if schedule.CoolDown != "" {
		_, err := time.ParseDuration(schedule.CoolDown)
		if err != nil {
			return nil, fmt.Errorf("invalid cooldown: %v", err)
		}
	}

	return schedule, nil
}

// findPrevScheduledTime finds the most recent time before now that matches the schedule
// I like 0 this convergent algorithm, but I'd rather use this and rely on a robust and maintained Next()
// than write my own Prev(). Plus performance is not that bad, it does converge fast enough.
func findPrevScheduledTime(schedule cron.Schedule, now time.Time) (time.Time, error) {
	if schedule == nil {
		return time.Time{}, nil
	}

	maxLookback := 7 * 24 * time.Hour

	startPoint := now.Add(-time.Second)

	minimum := now.Add(-maxLookback)
	maximum := now

	// Calculate the midpoint between now and minimum
	current := startPoint

	var lastFound time.Time

	shortCircuit := 30
	loopCount := 0

	for {
		loopCount++

		if loopCount > shortCircuit {
			return time.Time{}, fmt.Errorf("schedule prev() algorithm did not converge soon enough (or at all)")
		}

		next := schedule.Next(current)

		if next == now {
			return next, nil
		}

		if maximum.Sub(minimum) < 1*time.Minute {
			return lastFound, nil
		}

		if next.Before(now) {
			lastFound = next
			newCurrent := minimum.Add(maximum.Sub(minimum) / 2)
			minimum = current
			current = newCurrent
			continue
		}

		if next.After(now) {
			maximum = current
			newCurrent := current.Add(-current.Sub(minimum) / 2)
			current = newCurrent
			continue
		}
	}
}

// IsUpdateAllowed checks if an update is allowed based on the schedule and last update time
func (s *UpdateSchedule) IsUpdateAllowed(lastUpdateTime time.Time) (bool, error) {
	if s == nil {
		return true, nil
	}

	// Check cooldown period
	if s.CoolDown != "" {
		cooldown, _ := time.ParseDuration(s.CoolDown)
		if time.Since(lastUpdateTime) < cooldown {
			return false, nil
		}
	}

	// Check each cron schedule
	now := time.Now()

	for _, cronStr := range s.CronTabs {
		cronStr = strings.TrimSpace(cronStr)
		if cronStr == "" {
			continue
		}

		// Parse cron schedule
		schedule, err := cron.Parse(cronStr)
		if err != nil {
			return false, fmt.Errorf("invalid cron schedule '%s': %v", cronStr, err)
		}

		// Get previous run time
		prevRun, _ := findPrevScheduledTime(schedule, now)
		if prevRun.IsZero() {
			continue
		}

		// Check if we're within the update window
		if s.Duration != "" {
			duration, _ := time.ParseDuration(s.Duration)
			windowEnd := prevRun.Add(duration)
			if now.After(prevRun) && now.Before(windowEnd) {
				return true, nil
			}
		} else {
			// If no duration specified, only allow updates at the exact cron time
			if now.Equal(prevRun) {
				return true, nil
			}
		}
	}

	return false, nil
}
