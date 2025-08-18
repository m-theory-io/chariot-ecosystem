// tests/datetime_functions_test.go
package tests

import (
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

func TestDateTimeFunctions(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Current Timestamp",
			Script: []string{
				`setq(timestamp, now())`,
				`bigger(timestamp, '')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Today's Date",
			Script: []string{
				`setq(todayStr, today())`,
				`bigger(todayStr, '2020-01-01')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Date Creation - Single String",
			Script: []string{
				`date('2022-01-01')`,
			},
			ExpectedValue: chariot.Str("2022-01-01T00:00:00Z"),
		},
		{
			Name: "Date Creation - Components",
			Script: []string{
				`date(2022, 1, 15)`,
			},
			ExpectedValue: chariot.Str("2022-01-15T00:00:00Z"),
		},
		{
			Name: "Format Date - Default",
			Script: []string{
				`formatDate('2022-01-01T00:00:00Z')`,
			},
			ExpectedValue: chariot.Str("2022-01-01T00:00:00Z"),
		},
		{
			Name: "Format Date - Custom Format",
			Script: []string{
				`formatDate('2022-01-01T00:00:00Z', 'YYYY-MM-DD HH:mm:ss')`,
			},
			ExpectedValue: chariot.Str("2022-01-01 00:00:00"),
		},
		{
			Name: "Format Date - Date Only",
			Script: []string{
				`formatDate('2022-01-01T00:00:00Z', 'YYYY-MM-DD')`,
			},
			ExpectedValue: chariot.Str("2022-01-01"),
		},
		{
			Name: "Format Date - Time Only",
			Script: []string{
				`formatDate('2022-01-01T15:30:45Z', 'HH:mm:ss')`,
			},
			ExpectedValue: chariot.Str("15:30:45"),
		},
		{
			Name: "Add Days - Positive",
			Script: []string{
				`dateAdd('2022-01-01T00:00:00Z', 'days', 5)`,
			},
			ExpectedValue: chariot.Str("2022-01-06T00:00:00Z"),
		},
		{
			Name: "Add Days - Negative",
			Script: []string{
				`dateAdd('2022-01-01T00:00:00Z', 'days', -3)`,
			},
			ExpectedValue: chariot.Str("2021-12-29T00:00:00Z"),
		},
		{
			Name: "Add Months",
			Script: []string{
				`dateAdd('2022-01-01T00:00:00Z', 'months', 3)`,
			},
			ExpectedValue: chariot.Str("2022-04-01T00:00:00Z"),
		},
		{
			Name: "Add Years",
			Script: []string{
				`dateAdd('2022-01-01T00:00:00Z', 'years', 2)`,
			},
			ExpectedValue: chariot.Str("2024-01-01T00:00:00Z"),
		},
		{
			Name: "Add Hours",
			Script: []string{
				`dateAdd('2022-01-01T00:00:00Z', 'hours', 25)`,
			},
			ExpectedValue: chariot.Str("2022-01-02T01:00:00Z"),
		},
		{
			Name: "Date Difference - Days",
			Script: []string{
				`dateDiff('days', '2022-01-01T00:00:00Z', '2022-01-06T00:00:00Z')`,
			},
			ExpectedValue: chariot.Number(5),
		},
		{
			Name: "Date Difference - Hours",
			Script: []string{
				`dateDiff('hours', '2022-01-01T00:00:00Z', '2022-01-02T00:00:00Z')`,
			},
			ExpectedValue: chariot.Number(24),
		},
		{
			Name: "Date Difference - Minutes",
			Script: []string{
				`dateDiff('minutes', '2022-01-01T00:00:00Z', '2022-01-01T01:00:00Z')`,
			},
			ExpectedValue: chariot.Number(60),
		},
		{
			Name: "Date Components - Year",
			Script: []string{
				`year('2022-01-01T00:00:00Z')`,
			},
			ExpectedValue: chariot.Number(2022),
		},
		{
			Name: "Date Components - Month",
			Script: []string{
				`month('2022-03-15T00:00:00Z')`,
			},
			ExpectedValue: chariot.Number(3),
		},
		{
			Name: "Date Components - Day",
			Script: []string{
				`day('2022-01-15T00:00:00Z')`,
			},
			ExpectedValue: chariot.Number(15),
		},
		{
			Name: "Day of Week",
			Script: []string{
				`dayOfWeek('2022-01-03T00:00:00Z')`, // Monday
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Is Date - Valid",
			Script: []string{
				`isDate('2022-01-01T00:00:00Z')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Is Date - Invalid",
			Script: []string{
				`isDate('not-a-date')`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "Is Date - Non-string",
			Script: []string{
				`isDate(123)`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "Julian Day",
			Script: []string{
				`julianDay('2022-01-01T00:00:00Z')`,
			},
			ExpectedValue: chariot.Number(2459581),
		},
		{
			Name: "Year Fraction - Actual/365",
			Script: []string{
				`yearFraction('2022-01-01T00:00:00Z', '2022-01-31T00:00:00Z', 'actual/365')`,
			},
			ExpectedValue: chariot.Number(0.0821917808219178), // 30/365
		},
		{
			Name: "Is Business Day - Weekday",
			Script: []string{
				`isBusinessDay('2022-01-03T00:00:00Z')`, // Monday
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Is Business Day - Weekend",
			Script: []string{
				`isBusinessDay('2022-01-01T00:00:00Z')`, // Saturday
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "End of Month",
			Script: []string{
				`endOfMonth('2022-01-15T00:00:00Z')`,
			},
			ExpectedValue: chariot.Str("2022-01-31T00:00:00Z"),
		},
		{
			Name: "Is End of Month - True",
			Script: []string{
				`isEndOfMonth('2022-01-31T00:00:00Z')`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Is End of Month - False",
			Script: []string{
				`isEndOfMonth('2022-01-15T00:00:00Z')`,
			},
			ExpectedValue: chariot.Bool(false),
		},
		{
			Name: "Next Business Day",
			Script: []string{
				`nextBusinessDay('2022-01-07T00:00:00Z')`, // Friday -> Monday
			},
			ExpectedValue: chariot.Str("2022-01-10T00:00:00Z"),
		},
		{
			Name: "Current Date Operations",
			Script: []string{
				`setq(now1, now())`,
				`setq(tomorrow, dateAdd(now1, 'days', 1))`,
				`setq(diff, dateDiff('days', now1, tomorrow))`,
				`equal(diff, 1)`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Date Format Round Trip",
			Script: []string{
				`setq(original, '2022-01-01T15:30:45Z')`,
				`setq(formatted, formatDate(original, 'YYYY-MM-DD HH:mm:ss'))`,
				`setq(parsed, date(formatted))`,
				`equal(day(original), day(parsed))`,
			},
			ExpectedValue: chariot.Bool(true),
		},
	}

	RunTestCases(t, tests)
}

func TestDateTimeErrorHandling(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Date Creation - Invalid String",
			Script: []string{
				`date('invalid-date')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Date Creation - Invalid Arguments",
			Script: []string{
				`date(2022, 'invalid', 1)`,
			},
			ExpectedError: true,
		},
		{
			Name: "Format Date - Invalid Date",
			Script: []string{
				`formatDate('invalid-date', 'YYYY-MM-DD')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Format Date - Invalid Format",
			Script: []string{
				`formatDate('2022-01-01T00:00:00Z', 123)`,
			},
			ExpectedError: true,
		},
		{
			Name: "Date Add - Invalid Date",
			Script: []string{
				`dateAdd('invalid-date', 'days', 1)`,
			},
			ExpectedError: true,
		},
		{
			Name: "Date Add - Invalid Interval",
			Script: []string{
				`dateAdd('2022-01-01T00:00:00Z', 'invalid-interval', 1)`,
			},
			ExpectedError: true,
		},
		{
			Name: "Date Diff - Invalid Unit",
			Script: []string{
				`dateDiff('invalid-unit', '2022-01-01T00:00:00Z', '2022-01-02T00:00:00Z')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Date Diff - Wrong Argument Count",
			Script: []string{
				`dateDiff('days', '2022-01-01T00:00:00Z')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Year Fraction - Invalid Convention",
			Script: []string{
				`yearFraction('2022-01-01T00:00:00Z', '2022-01-31T00:00:00Z', 'invalid/convention')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Day Count - End Before Start",
			Script: []string{
				`dayCount('2022-01-31T00:00:00Z', '2022-01-01T00:00:00Z', 'actual/365')`,
			},
			ExpectedError: true,
		},
	}

	RunTestCases(t, tests)
}

// Add a test specifically for the parseDate function if you want to implement it
func TestParseDateFunction(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Parse Date to Unix Timestamp - ISO Format",
			Script: []string{
				`parseDate('2022-01-01T00:00:00Z')`,
			},
			ExpectedValue: chariot.Number(1640995200),
		},
		{
			Name: "Parse Date to Unix Timestamp - Custom Format",
			Script: []string{
				`parseDate('2022-01-01 00:00:00', 'YYYY-MM-DD HH:mm:ss')`,
			},
			ExpectedValue: chariot.Number(1640995200),
		},
		{
			Name: "Parse Date to Unix Timestamp - Date Only",
			Script: []string{
				`parseDate('2022-01-01', 'YYYY-MM-DD')`,
			},
			ExpectedValue: chariot.Number(1640995200),
		},
	}

	RunTestCases(t, tests)
}

// Add to datetime_functions_test.go
func TestLocalTimeConversion(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Local Time - Default Format",
			Script: []string{
				`setTimezone('America/New_York')`,
				`localTime('2022-01-01T12:00:00Z')`, // UTC noon
			},
			ExpectedValue: chariot.Str("2022-01-01T07:00:00-05:00"), // EST 7 AM
		},
		{
			Name: "Local Time - Custom Format",
			Script: []string{
				`setTimezone('America/New_York')`,
				`localTime('2022-06-01T12:00:00Z', 'YYYY-MM-DD HH:mm:ss')`, // UTC noon in summer
			},
			ExpectedValue: chariot.Str("2022-06-01 08:00:00"), // EDT 8 AM
		},
		{
			Name: "Local Time - Different Timezone",
			Script: []string{
				`setTimezone('Asia/Tokyo')`,
				`localTime('2022-01-01T12:00:00Z')`, // UTC noon
			},
			ExpectedValue: chariot.Str("2022-01-01T21:00:00+09:00"), // JST 9 PM
		},
		{
			Name: "UTC Time - Convert Local to UTC",
			Script: []string{
				`setTimezone('America/New_York')`,
				`utcTime('2022-01-01T07:00:00')`, // Local 7 AM EST
			},
			ExpectedValue: chariot.Str("2022-01-01T12:00:00Z"), // UTC noon
		},
		{
			Name: "Round Trip - UTC to Local to UTC",
			Script: []string{
				`setTimezone('America/Los_Angeles')`,
				`setq(original, '2022-01-01T15:30:00Z')`,
				`setq(local, localTime(original))`,
				`setq(backToUtc, utcTime(local))`,
				`equal(original, backToUtc)`,
			},
			ExpectedValue: chariot.Bool(true),
		},
		{
			Name: "Get Current Timezone",
			Script: []string{
				`setTimezone('Europe/London')`,
				`getTimezone()`,
			},
			ExpectedValue: chariot.Str("Europe/London"),
		},
		{
			Name: "Daylight Saving Time - Winter",
			Script: []string{
				`setTimezone('America/New_York')`,
				`localTime('2022-01-01T12:00:00Z')`, // Winter - EST
			},
			ExpectedValue: chariot.Str("2022-01-01T07:00:00-05:00"),
		},
		{
			Name: "Daylight Saving Time - Summer",
			Script: []string{
				`setTimezone('America/New_York')`,
				`localTime('2022-07-01T12:00:00Z')`, // Summer - EDT
			},
			ExpectedValue: chariot.Str("2022-07-01T08:00:00-04:00"),
		},
	}

	RunTestCases(t, tests)
}

func TestLocalTimeErrorHandling(t *testing.T) {
	tests := []TestCase{
		{
			Name: "Invalid Timezone",
			Script: []string{
				`setTimezone('Invalid/Timezone')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Local Time - Invalid UTC String",
			Script: []string{
				`localTime('invalid-date')`,
			},
			ExpectedError: true,
		},
		{
			Name: "UTC Time - Invalid Local String",
			Script: []string{
				`utcTime('invalid-date')`,
			},
			ExpectedError: true,
		},
		{
			Name: "Local Time - Wrong Argument Count",
			Script: []string{
				`localTime()`,
			},
			ExpectedError: true,
		},
	}

	RunTestCases(t, tests)
}
