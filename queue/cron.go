package queue

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Valentin-Kaiser/go-core/apperror"
)

// CronField represents a field in a cron expression
type CronField struct {
	Min, Max int
	Values   []int
}

// CronExpression represents a parsed cron expression
type CronExpression struct {
	Minute    CronField
	Hour      CronField
	Day       CronField
	Month     CronField
	DayOfWeek CronField
}

// validateCronSpec validates a cron specification
func (s *TaskScheduler) validateCronSpec(cronSpec string) error {
	_, err := s.parseCronSpec(cronSpec)
	return err
}

// parseCronSpec parses a cron specification
func (s *TaskScheduler) parseCronSpec(cronSpec string) (*CronExpression, error) {
	fields := strings.Fields(cronSpec)
	if len(fields) != 5 {
		return nil, apperror.NewError("cron expression must have exactly 5 fields (minute hour day month day-of-week)")
	}

	expr := &CronExpression{}
	var err error

	// Parse minute field (0-59)
	expr.Minute, err = s.parseCronField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field: %v", err)
	}

	// Parse hour field (0-23)
	expr.Hour, err = s.parseCronField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field: %v", err)
	}

	// Parse day field (1-31)
	expr.Day, err = s.parseCronField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day field: %v", err)
	}

	// Parse month field (1-12)
	expr.Month, err = s.parseCronField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field: %v", err)
	}

	// Parse day-of-week field (0-6, 0 = Sunday)
	expr.DayOfWeek, err = s.parseCronField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-week field: %v", err)
	}

	return expr, nil
}

// parseCronField parses a single field in a cron expression
func (s *TaskScheduler) parseCronField(field string, min, max int) (CronField, error) {
	cronField := CronField{Min: min, Max: max}

	if field == "*" {
		for i := min; i <= max; i++ {
			cronField.Values = append(cronField.Values, i)
		}
		return cronField, nil
	}

	if strings.Contains(field, "/") {
		parts := strings.Split(field, "/")
		if len(parts) != 2 {
			return cronField, apperror.NewError("invalid step format")
		}

		step, err := strconv.Atoi(parts[1])
		if err != nil || step <= 0 {
			return cronField, apperror.NewError("invalid step value")
		}

		var start, end int
		if parts[0] == "*" {
			start, end = min, max
		} else if strings.Contains(parts[0], "-") {
			rangeParts := strings.Split(parts[0], "-")
			if len(rangeParts) != 2 {
				return cronField, apperror.NewError("invalid range format")
			}
			start, err = strconv.Atoi(rangeParts[0])
			if err != nil || start < min || start > max {
				return cronField, apperror.NewError("invalid range start")
			}
			end, err = strconv.Atoi(rangeParts[1])
			if err != nil || end < min || end > max || end < start {
				return cronField, apperror.NewError("invalid range end")
			}
		} else {
			start, err = strconv.Atoi(parts[0])
			if err != nil || start < min || start > max {
				return cronField, apperror.NewError("invalid step start")
			}
			end = max
		}

		for i := start; i <= end; i += step {
			cronField.Values = append(cronField.Values, i)
		}
		return cronField, nil
	}

	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) != 2 {
			return cronField, apperror.NewError("invalid range format")
		}

		start, err := strconv.Atoi(parts[0])
		if err != nil || start < min || start > max {
			return cronField, apperror.NewError("invalid range start")
		}

		end, err := strconv.Atoi(parts[1])
		if err != nil || end < min || end > max || end < start {
			return cronField, apperror.NewError("invalid range end")
		}

		for i := start; i <= end; i++ {
			cronField.Values = append(cronField.Values, i)
		}
		return cronField, nil
	}

	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		for _, part := range parts {
			value, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || value < min || value > max {
				return cronField, apperror.NewError("invalid value in list")
			}
			cronField.Values = append(cronField.Values, value)
		}
		return cronField, nil
	}

	value, err := strconv.Atoi(field)
	if err != nil || value < min || value > max {
		return cronField, apperror.NewError("invalid single value")
	}
	cronField.Values = append(cronField.Values, value)

	return cronField, nil
}

// calculateNextCronRun calculates the next run time for a cron expression
func (s *TaskScheduler) calculateNextCronRun(cronSpec string, after time.Time) (time.Time, error) {
	expr, err := s.parseCronSpec(cronSpec)
	if err != nil {
		return time.Time{}, err
	}

	t := after.Truncate(time.Minute).Add(time.Minute)

	// Find the next matching time (within reasonable limits)
	for attempts := 0; attempts < 366*24*60; attempts++ { // Max 1 year
		if s.cronMatches(expr, t) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}

	return time.Time{}, apperror.NewError("could not find next run time within reasonable limits")
}

// cronMatches checks if a time matches a cron expression
func (s *TaskScheduler) cronMatches(expr *CronExpression, t time.Time) bool {
	if !s.fieldMatches(expr.Minute, t.Minute()) {
		return false
	}

	if !s.fieldMatches(expr.Hour, t.Hour()) {
		return false
	}

	if !s.fieldMatches(expr.Month, int(t.Month())) {
		return false
	}

	if !s.fieldMatches(expr.DayOfWeek, int(t.Weekday())) {
		return false
	}

	if !s.fieldMatches(expr.Day, t.Day()) {
		return false
	}

	return true
}

// fieldMatches checks if a value matches a cron field
func (s *TaskScheduler) fieldMatches(field CronField, value int) bool {
	for _, v := range field.Values {
		if v == value {
			return true
		}
	}
	return false
}
