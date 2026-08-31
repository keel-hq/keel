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
	Schedules []Schedule
}

// Schedule represents a single schedule entry with its window duration
type Schedule struct {
	Crontab  cron.Schedule
	Duration time.Duration
}

// ParseUpdateSchedule parses the update schedule from annotations
func ParseUpdateSchedule(annotations map[string]string) (*UpdateSchedule, error) {
	if annotations == nil {
		return nil, nil
	}

	cronStr := annotations[types.KeelUpdateScheduleCronTabs]
	if cronStr == "" {
		return nil, nil
	}

	// Parse and validate cron schedules
	cronStrings := strings.Split(cronStr, ",")
	var schedules []Schedule

	for _, cs := range cronStrings {
		cs = strings.TrimSpace(cs)
		if cs == "" {
			continue
		}

		// Split into crontab and duration parts
		parts := strings.Split(cs, "|")
		if len(parts) != 2 {
			return nil, fmt.Errorf("internal.schedule.schedule: invalid schedule format '%s', expected 'CRONTAB|DURATION,CRONTAB2|DURATION2'", cs)
		}

		// Parse crontab
		cronPart := strings.TrimSpace(parts[0])
		schedule, err := cron.Parse(cronPart)
		if err != nil {
			return nil, fmt.Errorf("internal.schedule.schedule: invalid cron schedule '%s': %v", cronPart, err)
		}

		// Parse duration if provided
		var duration time.Duration
		durationStr := strings.TrimSpace(parts[1])
		if durationStr != "" {
			duration, err = time.ParseDuration(durationStr)
			if err != nil {
				return nil, fmt.Errorf("internal.schedule.schedule: invalid duration '%s': %v", durationStr, err)
			}
		}

		schedules = append(schedules, Schedule{
			Crontab:  schedule,
			Duration: duration,
		})
	}

	// If no valid schedules found, return nil
	if len(schedules) == 0 {
		return nil, nil
	}

	return &UpdateSchedule{
		Schedules: schedules,
	}, nil
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
			return time.Time{}, fmt.Errorf("internal.schedule.schedule prev() algorithm did not converge soon enough (or at all)")
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
func (s *UpdateSchedule) IsUpdateAllowed(lastUpdateTime time.Time, now time.Time) (bool, error) {
	if s == nil {
		return true, nil
	}

	for _, schedule := range s.Schedules {
		// Get previous run time
		prevRun, err := findPrevScheduledTime(schedule.Crontab, now)
		if err != nil {
			continue
		}
		if prevRun.IsZero() {
			continue
		}

		// Check if we're within cooldown period
		if now.Sub(lastUpdateTime) < schedule.Duration {
			continue
		}

		// Check if we're within the update window
		windowEnd := prevRun.Add(schedule.Duration)
		if now.Equal(prevRun) || (now.After(prevRun) && now.Before(windowEnd)) {
			return true, nil
		}
	}

	return false, nil
}
