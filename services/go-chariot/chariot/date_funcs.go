package chariot

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Add to date_funcs.go at the top with other constants
const CHARIOT_DATETIME_FORMAT = "2006-01-02T15:04:05Z"

// Common timezone constants for testing
const (
	TZ_UTC      = "UTC"
	TZ_EASTERN  = "America/New_York"
	TZ_CENTRAL  = "America/Chicago"
	TZ_MOUNTAIN = "America/Denver"
	TZ_PACIFIC  = "America/Los_Angeles"
	TZ_LONDON   = "Europe/London"
	TZ_TOKYO    = "Asia/Tokyo"
	TZ_SYDNEY   = "Australia/Sydney"
)

// Default timezone for local time conversion (will be replaced by user profile setting)
var LocalTimezone = "America/Los_Angeles" // Can be changed to test different timezones

// RegisterDate registers all date-related functions
func RegisterDate(rt *Runtime) {
	// Date creation
	rt.Register("now", func(args ...Value) (Value, error) {
		// Return current time as string in standard format
		return Str(time.Now().UTC().Format(CHARIOT_DATETIME_FORMAT)), nil
	})

	// parseDate function to extract UNIX timestamp from string
	rt.Register("parseDate", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("parseDate requires 1 or 2 arguments")
		}

		dateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("expected string date, got %T", args[0])
		}

		var t time.Time
		var err error
		if len(args) == 2 {
			formatStr, ok := args[1].(Str)
			if !ok {
				return nil, fmt.Errorf("expected string format, got %T", args[1])
			}
			t, err = parseDateFormat(string(dateStr), string(formatStr))
		} else {
			t, err = parseDate(string(dateStr))
		}

		if err != nil {
			return nil, err
		}

		return Number(t.Unix()), nil
	})

	rt.Register("today", func(args ...Value) (Value, error) {
		// Return current date as string in YYYY-MM-DD format
		now := time.Now()
		return Str(fmt.Sprintf("%04d-%02d-%02d", now.Year(), now.Month(), now.Day())), nil
	})

	rt.Register("localTime", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("localTime requires 1 or 2 arguments: utcDateTime [, format]")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Parse UTC datetime string
		utcDateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("utcDateTime must be a string, got %T", args[0])
		}

		utcTime, err := parseDate(string(utcDateStr))
		if err != nil {
			return nil, fmt.Errorf("could not parse UTC datetime: %v", err)
		}

		// Load the local timezone
		location, err := time.LoadLocation(LocalTimezone)
		if err != nil {
			return nil, fmt.Errorf("could not load timezone %s: %v", LocalTimezone, err)
		}

		// Convert to local time
		localTime := utcTime.In(location)

		// Format output
		var result string
		if len(args) == 2 {
			// Custom format provided
			formatStr, ok := args[1].(Str)
			if !ok {
				return nil, fmt.Errorf("format must be a string, got %T", args[1])
			}

			// Convert format to Go format
			goFormat := translateTimeFormat(string(formatStr))
			result = localTime.Format(goFormat)
		} else {
			// Default format - RFC3339 with timezone
			result = localTime.Format(time.RFC3339)
		}

		return Str(result), nil
	})

	rt.Register("date", func(args ...Value) (Value, error) {
		if len(args) == 0 {
			return nil, errors.New("date requires at least 1 argument")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// If single string argument, try to parse it
		if len(args) == 1 {
			if str, ok := args[0].(Str); ok {
				t, err := parseDate(string(str))
				if err != nil {
					return nil, err
				}
				return Str(t.UTC().Format(CHARIOT_DATETIME_FORMAT)), nil
			}
			return nil, fmt.Errorf("expected string date, got %T", args[0])
		}

		// If 3 arguments (year, month, day)
		if len(args) == 3 {
			year, yearOk := args[0].(Number)
			month, monthOk := args[1].(Number)
			day, dayOk := args[2].(Number)

			if !yearOk || !monthOk || !dayOk {
				return nil, errors.New("year, month, and day must all be numbers")
			}

			// Create time from components
			t := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
			return Str(t.Format(CHARIOT_DATETIME_FORMAT)), nil
		}

		return nil, errors.New("date requires either 1 string argument or 3 numeric arguments (year, month, day)")
	})

	// Date manipulation
	rt.Register("dateAdd", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("dateAdd requires 3 arguments: date, interval, value")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Parse base date
		dateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("date must be a string, got %T", args[0])
		}

		baseDate, err := parseDate(string(dateStr))
		if err != nil {
			return nil, err
		}

		// Get interval type
		intervalType, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("interval type must be a string, got %T", args[1])
		}

		// Get interval value
		value, ok := args[2].(Number)
		if !ok {
			return nil, fmt.Errorf("interval value must be a number, got %T", args[2])
		}

		// Add interval to date
		var resultDate time.Time

		switch strings.ToLower(string(intervalType)) {
		case "year", "years":
			resultDate = baseDate.AddDate(int(value), 0, 0)
		case "month", "months":
			resultDate = baseDate.AddDate(0, int(value), 0)
		case "day", "days":
			resultDate = baseDate.AddDate(0, 0, int(value))
		case "hour", "hours":
			resultDate = baseDate.Add(time.Duration(value) * time.Hour)
		case "minute", "minutes":
			resultDate = baseDate.Add(time.Duration(value) * time.Minute)
		case "second", "seconds":
			resultDate = baseDate.Add(time.Duration(value) * time.Second)
		default:
			return nil, fmt.Errorf("unknown interval type: %s", intervalType)
		}

		return Str(resultDate.Format(time.RFC3339)), nil
	})

	rt.Register("dateDiff", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("dateDiff requires 3 arguments: interval, date1, date2")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get interval type
		intervalType, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("interval type must be a string, got %T", args[0])
		}

		// Parse first date
		date1Str, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("date1 must be a string, got %T", args[1])
		}

		date1, err := parseDate(string(date1Str))
		if err != nil {
			return nil, err
		}

		// Parse second date
		date2Str, ok := args[2].(Str)
		if !ok {
			return nil, fmt.Errorf("date2 must be a string, got %T", args[2])
		}

		date2, err := parseDate(string(date2Str))
		if err != nil {
			return nil, err
		}

		// Calculate difference based on interval
		var diff float64

		switch strings.ToLower(string(intervalType)) {
		case "year", "years":
			diff = float64(date2.Year() - date1.Year())
		case "month", "months":
			yearDiff := date2.Year() - date1.Year()
			monthDiff := int(date2.Month()) - int(date1.Month())
			diff = float64(yearDiff*12 + monthDiff)
		case "day", "days":
			diff = date2.Sub(date1).Hours() / 24
		case "hour", "hours":
			diff = date2.Sub(date1).Hours()
		case "minute", "minutes":
			diff = date2.Sub(date1).Minutes()
		case "second", "seconds":
			diff = date2.Sub(date1).Seconds()
		default:
			return nil, fmt.Errorf("unknown interval type: %s", intervalType)
		}

		return Number(diff), nil
	})

	// Date components
	rt.Register("day", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("day requires 1 argument")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		dateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("date must be a string, got %T", args[0])
		}

		date, err := parseDate(string(dateStr))
		if err != nil {
			return nil, err
		}

		return Number(date.Day()), nil
	})
	rt.Register("dayOfWeek", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("dayOfWeek requires 1 argument")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		dateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("date must be a string, got %T", args[0])
		}
		date, err := parseDate(string(dateStr))
		if err != nil {
			return nil, err
		}
		return Number(int(date.Weekday())), nil
	})

	rt.Register("month", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("month requires 1 argument")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		dateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("date must be a string, got %T", args[0])
		}

		date, err := parseDate(string(dateStr))
		if err != nil {
			return nil, err
		}

		return Number(int(date.Month())), nil
	})

	rt.Register("year", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("year requires 1 argument")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		dateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("date must be a string, got %T", args[0])
		}

		date, err := parseDate(string(dateStr))
		if err != nil {
			return nil, err
		}

		return Number(date.Year()), nil
	})

	rt.Register("julianDay", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("julianDay requires 1 argument")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		dateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("date must be a string, got %T", args[0])
		}

		date, err := parseDate(string(dateStr))
		if err != nil {
			return nil, err
		}

		// Calculate Julian day number
		// Formula from: https://en.wikipedia.org/wiki/Julian_day
		a := (14 - int(date.Month())) / 12
		y := date.Year() + 4800 - a
		m := int(date.Month()) + 12*a - 3

		jdn := date.Day() + (153*m+2)/5 + 365*y + y/4 - y/100 + y/400 - 32045

		return Number(jdn), nil
	})

	// Date utilities
	rt.Register("isDate", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("isDate requires 1 argument")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// If not a string, it's not a date
		str, ok := args[0].(Str)
		if !ok {
			return Bool(false), nil
		}

		// Try to parse the date
		_, err := parseDate(string(str))

		// Return whether parsing succeeded
		return Bool(err == nil), nil
	})

	rt.Register("formatDate", func(args ...Value) (Value, error) {
		if len(args) > 2 || len(args) < 1 {
			return nil, errors.New("formatDate requires 1 or 2 arguments: date and optionally format")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Get the date
		dateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("date must be a string, got %T", args[0])
		}

		date, err := parseDate(string(dateStr))
		if err != nil {
			return nil, err
		}

		formatStr := CHARIOT_DATETIME_FORMAT

		// Get the user's format pattern
		if len(args) > 1 {
			if tvar, ok := args[1].(Str); ok {
				formatStr = string(tvar)
			} else {
				return nil, fmt.Errorf("format must be a string, got %T", args[1])
			}
		}

		// Convert format string to Go format
		goFormat := translateTimeFormat(string(formatStr))

		// Format the date
		result := date.Format(goFormat)

		return Str(result), nil
	})

	rt.Register("dayCount", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("dayCount requires 3 arguments: startDate, endDate, convention")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		startStr, ok1 := args[0].(Str)
		endStr, ok2 := args[1].(Str)
		convention, ok3 := args[2].(Str)
		if !ok1 || !ok2 || !ok3 {
			return nil, errors.New("dayCount requires (string, string, string)")
		}
		start, err := parseDate(string(startStr))
		if err != nil {
			return nil, fmt.Errorf("could not parse startDate: %v", err)
		}
		end, err := parseDate(string(endStr))
		if err != nil {
			return nil, fmt.Errorf("could not parse endDate: %v", err)
		}
		if end.Before(start) {
			return nil, errors.New("endDate must not be before startDate")
		}

		switch strings.ToLower(string(convention)) {
		case "actual/360":
			days := end.Sub(start).Hours() / 24
			return Number(days / 360.0), nil
		case "actual/365":
			days := end.Sub(start).Hours() / 24
			return Number(days / 365.0), nil
		case "actual/actual":
			days := end.Sub(start).Hours() / 24
			// Use the actual number of days in the year of the start date
			yearDays := 365.0
			year := start.Year()
			if (year%4 == 0 && year%100 != 0) || (year%400 == 0) {
				yearDays = 366.0
			}
			return Number(days / yearDays), nil
		case "30/360":
			// 30/360 US convention
			y1, m1, d1 := start.Date()
			y2, m2, d2 := end.Date()
			if d1 == 31 {
				d1 = 30
			}
			if d2 == 31 && d1 == 30 {
				d2 = 30
			}
			days := 360*(y2-y1) + 30*(int(m2)-int(m1)) + (d2 - d1)
			return Number(float64(days) / 360.0), nil
		default:
			return nil, fmt.Errorf("unsupported day count convention: %s", convention)
		}
	})
	rt.Register("yearFraction", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("yearFraction requires 3 arguments: startDate, endDate, convention")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		startStr, ok1 := args[0].(Str)
		endStr, ok2 := args[1].(Str)
		convention, ok3 := args[2].(Str)
		if !ok1 || !ok2 || !ok3 {
			return nil, errors.New("yearFraction requires (string, string, string)")
		}
		start, err := parseDate(string(startStr))
		if err != nil {
			return nil, fmt.Errorf("could not parse startDate: %v", err)
		}
		end, err := parseDate(string(endStr))
		if err != nil {
			return nil, fmt.Errorf("could not parse endDate: %v", err)
		}
		if end.Before(start) {
			return nil, errors.New("endDate must not be before startDate")
		}

		switch strings.ToLower(string(convention)) {
		case "actual/360":
			days := end.Sub(start).Hours() / 24
			return Number(days / 360.0), nil
		case "actual/365":
			days := end.Sub(start).Hours() / 24
			return Number(days / 365.0), nil
		case "actual/actual":
			days := end.Sub(start).Hours() / 24
			yearDays := 365.0
			year := start.Year()
			if (year%4 == 0 && year%100 != 0) || (year%400 == 0) {
				yearDays = 366.0
			}
			return Number(days / yearDays), nil
		case "30/360":
			y1, m1, d1 := start.Date()
			y2, m2, d2 := end.Date()
			if d1 == 31 {
				d1 = 30
			}
			if d2 == 31 && d1 == 30 {
				d2 = 30
			}
			days := 360*(y2-y1) + 30*(int(m2)-int(m1)) + (d2 - d1)
			return Number(float64(days) / 360.0), nil
		default:
			return nil, fmt.Errorf("unsupported day count convention: %s", convention)
		}
	})
	rt.Register("isBusinessDay", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("isBusinessDay requires 1 or 2 arguments: date [, holidays]")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		dateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("date must be a string, got %T", args[0])
		}
		date, err := parseDate(string(dateStr))
		if err != nil {
			return nil, err
		}
		weekday := date.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			return Bool(false), nil
		}
		// Check holidays if provided
		if len(args) == 2 {
			holidays, ok := args[1].(*ArrayValue)
			if !ok {
				return nil, errors.New("holidays must be an array of strings")
			}
			for i := 0; i < holidays.Length(); i++ {
				holidayStr, ok := holidays.Get(i).(Str)
				if !ok {
					continue
				}
				holiday, err := parseDate(string(holidayStr))
				if err == nil && date.Year() == holiday.Year() && date.YearDay() == holiday.YearDay() {
					return Bool(false), nil
				}
			}
		}
		return Bool(true), nil
	})

	rt.Register("nextBusinessDay", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("nextBusinessDay requires 1 or 2 arguments: date [, holidays]")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		dateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("date must be a string, got %T", args[0])
		}
		date, err := parseDate(string(dateStr))
		if err != nil {
			return nil, err
		}

		// Prepare holidays set for fast lookup
		holidaySet := make(map[string]struct{})
		if len(args) == 2 {
			holidays, ok := args[1].(*ArrayValue)
			if !ok {
				return nil, errors.New("holidays must be an array of strings")
			}
			for i := 0; i < holidays.Length(); i++ {
				holidayStr, ok := holidays.Get(i).(Str)
				if ok {
					holidayDate, err := parseDate(string(holidayStr))
					if err == nil {
						holidaySet[holidayDate.Format("2006-01-02")] = struct{}{}
					}
				}
			}
		}

		// Find the next business day
		for {
			date = date.AddDate(0, 0, 1)
			weekday := date.Weekday()
			if weekday == time.Saturday || weekday == time.Sunday {
				continue
			}
			if _, isHoliday := holidaySet[date.Format("2006-01-02")]; isHoliday {
				continue
			}
			break
		}

		return Str(date.Format(time.RFC3339)), nil
	})

	rt.Register("endOfMonth", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("endOfMonth requires 1 argument: date")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		dateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("date must be a string, got %T", args[0])
		}
		date, err := parseDate(string(dateStr))
		if err != nil {
			return nil, err
		}
		// Move to the first day of the next month, then subtract one day
		firstOfNextMonth := date.AddDate(0, 1, -date.Day()+1)
		endOfMonth := firstOfNextMonth.AddDate(0, 0, -1)
		return Str(endOfMonth.Format(time.RFC3339)), nil
	})

	rt.Register("isEndOfMonth", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("isEndOfMonth requires 1 argument: date")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		dateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("date must be a string, got %T", args[0])
		}
		date, err := parseDate(string(dateStr))
		if err != nil {
			return nil, err
		}
		// Move to the next day
		nextDay := date.AddDate(0, 0, 1)
		// If the next day is the first of the next month, then date is end of month
		isEnd := nextDay.Month() != date.Month()
		return Bool(isEnd), nil
	})

	rt.Register("dateSchedule", func(args ...Value) (Value, error) {
		if len(args) < 3 || len(args) > 4 {
			return nil, errors.New("dateSchedule requires 3 or 4 arguments: startDate, n, interval [, options]")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		startStr, ok1 := args[0].(Str)
		n, ok2 := args[1].(Number)
		interval, ok3 := args[2].(Str)
		if !ok1 || !ok2 || !ok3 {
			return nil, errors.New("dateSchedule requires (string, number, string [, options])")
		}
		start, err := parseDate(string(startStr))
		if err != nil {
			return nil, fmt.Errorf("could not parse startDate: %v", err)
		}
		count := int(n)
		if count < 1 {
			return nil, errors.New("n must be at least 1")
		}

		// Parse options
		businessDays := false
		endOfMonth := false
		holidaySet := map[string]struct{}{}
		if len(args) == 4 {
			opts, ok := args[3].(map[string]Value)
			if ok {
				if bd, ok := opts["businessDays"].(Bool); ok {
					businessDays = bool(bd)
				}
				if eom, ok := opts["endOfMonth"].(Bool); ok {
					endOfMonth = bool(eom)
				}
				if holidays, ok := opts["holidays"].(*ArrayValue); ok {
					for i := 0; i < holidays.Length(); i++ {
						holidayStr, ok := holidays.Get(i).(Str)
						if ok {
							holidayDate, err := parseDate(string(holidayStr))
							if err == nil {
								holidaySet[holidayDate.Format("2006-01-02")] = struct{}{}
							}
						}
					}
				}
			}
		}

		schedule := &ArrayValue{}
		for i := 0; i < count; i++ {
			var d time.Time
			switch strings.ToLower(string(interval)) {
			case "day", "days":
				d = start.AddDate(0, 0, i)
			case "week", "weeks":
				d = start.AddDate(0, 0, i*7)
			case "month", "months":
				d = start.AddDate(0, i, 0)
			case "year", "years":
				d = start.AddDate(i, 0, 0)
			default:
				return nil, fmt.Errorf("unsupported interval: %s", interval)
			}

			// End-of-month alignment
			if endOfMonth {
				// Move to the last day of the month
				firstOfNextMonth := d.AddDate(0, 1, -d.Day()+1)
				d = firstOfNextMonth.AddDate(0, 0, -1)
			}

			// Business day adjustment
			if businessDays {
				for {
					weekday := d.Weekday()
					if weekday == time.Saturday || weekday == time.Sunday {
						d = d.AddDate(0, 0, 1)
						continue
					}
					if _, isHoliday := holidaySet[d.Format("2006-01-02")]; isHoliday {
						d = d.AddDate(0, 0, 1)
						continue
					}
					break
				}
			}

			schedule.Append(Str(d.Format(time.RFC3339)))
		}
		return schedule, nil
	})

	rt.Register("setTimezone", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("setTimezone requires 1 argument: timezone")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		timezoneStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("timezone must be a string, got %T", args[0])
		}

		err := SetUserTimezone(string(timezoneStr))
		if err != nil {
			return nil, err
		}

		return Str(string(timezoneStr)), nil
	})

	rt.Register("getTimezone", func(args ...Value) (Value, error) {
		if len(args) != 0 {
			return nil, errors.New("getTimezone requires no arguments")
		}

		return Str(GetUserTimezone()), nil
	})

	rt.Register("utcTime", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("utcTime requires 1 or 2 arguments: localDateTime [, format]")
		}

		// Unwrap arguments
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		// Parse local datetime string
		localDateStr, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("localDateTime must be a string, got %T", args[0])
		}

		// Load the local timezone
		location, err := time.LoadLocation(LocalTimezone)
		if err != nil {
			return nil, fmt.Errorf("could not load timezone %s: %v", LocalTimezone, err)
		}

		var localTime time.Time
		if len(args) == 2 {
			// Custom format provided
			formatStr, ok := args[1].(Str)
			if !ok {
				return nil, fmt.Errorf("format must be a string, got %T", args[1])
			}

			goFormat := translateTimeFormat(string(formatStr))
			localTime, err = time.ParseInLocation(goFormat, string(localDateStr), location)
		} else {
			// Try to parse with various formats, assuming local timezone
			localTime, err = time.ParseInLocation(time.RFC3339, string(localDateStr), location)
			if err != nil {
				// Try without timezone info (assume local)
				localTime, err = time.ParseInLocation("2006-01-02T15:04:05", string(localDateStr), location)
			}
		}

		if err != nil {
			return nil, fmt.Errorf("could not parse local datetime: %v", err)
		}

		// Convert to UTC and format
		utcTime := localTime.UTC()
		result := utcTime.Format(CHARIOT_DATETIME_FORMAT)

		return Str(result), nil
	})
}

// Helper function to parse dates in various formats
func parseDate(dateStr string) (time.Time, error) {
	// Try standard formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006",
		"01/02/2006 15:04:05",
		"2006/01/02",
		"2006/01/02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	// Try to detect format from the string
	// This is a simplified version - would need more patterns in a real implementation
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}`, dateStr); matched {
		// YYYY-MM-DD format (possibly with time)
		if t, err := time.Parse("2006-01-02", dateStr[:10]); err == nil {
			return t, nil
		}
	} else if matched, _ := regexp.MatchString(`^\d{1,2}/\d{1,2}/\d{4}`, dateStr); matched {
		// MM/DD/YYYY format
		parts := strings.Split(dateStr, "/")
		if len(parts) >= 3 {
			month, _ := strconv.Atoi(parts[0])
			day, _ := strconv.Atoi(parts[1])
			year, _ := strconv.Atoi(parts[2])

			return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date: %s", dateStr)
}

func parseDateFormat(dateStr string, format string) (time.Time, error) {
	// Convert the custom format to Go's time format
	goFormat := translateTimeFormat(format)

	// Parse the date string using the converted format
	t, err := time.Parse(goFormat, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse date %s with format %s: %v", dateStr, goFormat, err)
	}

	return t, nil
}

// Helper function to translate common time format patterns to Go's time format
func translateTimeFormat(format string) string {
	// This is a simplified translation - order matters!
	// Replace longer patterns first to avoid conflicts
	replacements := []struct {
		pattern     string
		replacement string
	}{
		{"YYYY", "2006"}, // 4-digit year - must come before YY
		{"YY", "06"},     // 2-digit year
		{"MM", "01"},     // Month with leading zero
		{"DD", "02"},     // Day with leading zero
		{"HH", "15"},     // Hour in 24-hour format
		{"mm", "04"},     // Minute with leading zero
		{"ss", "05"},     // Second with leading zero
	}

	result := format
	for _, repl := range replacements {
		result = strings.ReplaceAll(result, repl.pattern, repl.replacement)
	}

	return result
}

// SetUserTimezone sets the timezone for the current user session
// This will eventually be replaced by user profile data
func SetUserTimezone(timezone string) error {
	// Validate timezone
	_, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone: %v", err)
	}

	LocalTimezone = timezone
	return nil
}

// GetUserTimezone returns the current user's timezone
func GetUserTimezone() string {
	return LocalTimezone
}
