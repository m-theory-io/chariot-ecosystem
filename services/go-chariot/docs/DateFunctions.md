# Chariot Language Reference

## Date Functions

Chariot provides a comprehensive set of date and time functions for creation, manipulation, formatting, timezone conversion, and financial calculations. Dates are represented as strings in ISO 8601 format (`YYYY-MM-DDTHH:MM:SSZ`), unless otherwise noted.

### Available Date Functions

| Function                | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `now()`                 | Current date and time as RFC3339 string (UTC)                    |
| `today()`               | Current date as `YYYY-MM-DD` string (local time)                 |
| `date(str)`             | Parse a date string to RFC3339 format                            |
| `date(year, month, day)`| Create a date from year, month, day (numbers)                    |
| `dateAdd(date, interval, value)` | Add interval (`year`, `month`, `day`, etc.) to date      |
| `dateDiff(interval, date1, date2)` | Difference between two dates in given interval         |
| `day(date)`             | Day of the month (1–31)                                          |
| `dayOfWeek(date)`       | Day of the week (0=Sunday, 6=Saturday)                           |
| `month(date)`           | Month number (1–12)                                              |
| `year(date)`            | Year number (e.g., 2025)                                         |
| `julianDay(date)`       | Julian day number                                                |
| `isDate(str)`           | Returns `true` if string is a valid date                         |
| `formatDate(date, [format])` | Format date using custom pattern (default RFC3339)           |
| `localTime(utcDateTime, [format])` | Convert UTC datetime to local time string              |
| `utcTime(localDateTime, [format])` | Convert local time string to UTC datetime              |
| `setTimezone(timezone)` | Set the default timezone for conversions                         |
| `getTimezone()`         | Get the current default timezone                                 |
| `dayCount(start, end, convention)` | Day count fraction between dates (financial)           |
| `yearFraction(start, end, convention)` | Year fraction between dates (financial)            |
| `isBusinessDay(date [, holidays])` | Returns `true` if date is a business day               |
| `nextBusinessDay(date [, holidays])` | Returns next business day after date                 |
| `endOfMonth(date)`      | Returns the last day of the month for the given date             |
| `isEndOfMonth(date)`    | Returns `true` if date is the last day of the month              |
| `dateSchedule(start, n, interval [, options])` | Generate a schedule of dates                |
| `parseDate(str, [format])` | Parse a date string to UNIX timestamp (seconds)               |

---

### Function Details

#### `now()`

Returns the current date and time as a string in RFC3339 format (UTC).

```chariot
now()  // "2025-06-27T14:23:00Z"
```

#### `today()`

Returns the current date as a string in `YYYY-MM-DD` format (local time).

```chariot
today()  // "2025-06-27"
```

#### `date(str)`  
Parses a date string and returns it in RFC3339 format. Accepts many common date formats.

```chariot
date("2025-06-27")         // "2025-06-27T00:00:00Z"
date("06/27/2025")         // "2025-06-27T00:00:00Z"
```

#### `date(year, month, day)`

Creates a date from numeric year, month, and day.

```chariot
date(2025, 6, 27)          // "2025-06-27T00:00:00Z"
```

#### `dateAdd(date, interval, value)`

Adds an interval to a date. Interval can be `"year"`, `"month"`, `"day"`, `"hour"`, `"minute"`, or `"second"` (singular or plural).

```chariot
dateAdd("2025-06-27", "day", 5)      // "2025-07-02T00:00:00Z"
dateAdd("2025-06-27", "month", 1)    // "2025-07-27T00:00:00Z"
```

#### `dateDiff(interval, date1, date2)`

Returns the difference between `date1` and `date2` in the specified interval.

```chariot
dateDiff("day", "2025-06-27", "2025-07-02")   // 5
dateDiff("month", "2025-01-01", "2025-06-01") // 5
```

#### `day(date)`

Returns the day of the month (1–31).

```chariot
day("2025-06-27")   // 27
```

#### `dayOfWeek(date)`

Returns the day of the week as a number (0=Sunday, 6=Saturday).

```chariot
dayOfWeek("2025-06-27")  // 5 (Friday)
```

#### `month(date)`

Returns the month number (1–12).

```chariot
month("2025-06-27")  // 6
```

#### `year(date)`

Returns the year as a number.

```chariot
year("2025-06-27")   // 2025
```

#### `julianDay(date)`

Returns the Julian day number for the given date.

```chariot
julianDay("2025-06-27")  // e.g., 2460492
```

#### `isDate(str)`

Returns `true` if the string is a valid date.

```chariot
isDate("2025-06-27")   // true
isDate("not a date")   // false
```

#### `formatDate(date, [format])`

Formats a date using a custom pattern.  
Supported patterns:  
- `YYYY` = year (e.g., 2025)
- `YY`   = two-digit year
- `MM`   = month (01–12)
- `DD`   = day (01–31)
- `HH`   = hour (00–23)
- `mm`   = minute (00–59)
- `ss`   = second (00–59)

If `format` is omitted, returns RFC3339.

```chariot
formatDate("2025-06-27T15:04:05Z", "YYYY/MM/DD HH:mm:ss") // "2025/06/27 15:04:05"
```

#### `localTime(utcDateTime, [format])`

Converts a UTC datetime string to the configured local timezone.  
If `format` is provided, uses the custom pattern.

```chariot
localTime("2025-06-27T15:04:05Z") // "2025-06-27T08:04:05-07:00" (if PDT)
localTime("2025-06-27T15:04:05Z", "YYYY-MM-DD HH:mm") // "2025-06-27 08:04"
```

#### `utcTime(localDateTime, [format])`

Converts a local time string to UTC datetime in standard format.  
If `format` is provided, parses using the custom pattern.

```chariot
utcTime("2025-06-27 08:04:05", "YYYY-MM-DD HH:mm:ss") // "2025-06-27T15:04:05Z"
```

#### `setTimezone(timezone)`

Sets the default timezone for conversions (e.g., `"America/Los_Angeles"`).

```chariot
setTimezone("America/New_York")
```

#### `getTimezone()`

Returns the current default timezone.

```chariot
getTimezone() // "America/New_York"
```

#### `dayCount(start, end, convention)`

Returns the day count fraction between two dates using a financial convention.  
Supported conventions: `"actual/360"`, `"actual/365"`, `"actual/actual"`, `"30/360"`

```chariot
dayCount("2025-01-01", "2025-07-01", "actual/360")
```

#### `yearFraction(start, end, convention)`

Returns the year fraction between two dates using a financial convention.  
Supported conventions: `"actual/360"`, `"actual/365"`, `"actual/actual"`, `"30/360"`

```chariot
yearFraction("2025-01-01", "2025-07-01", "actual/365")
```

#### `isBusinessDay(date [, holidays])`

Returns `true` if the date is a business day (not a weekend or holiday).  
`holidays` is an optional array of date strings.

```chariot
isBusinessDay("2025-06-27") // true (if not a weekend/holiday)
isBusinessDay("2025-06-28") // false (Saturday)
isBusinessDay("2025-07-04", array("2025-07-04")) // false (holiday)
```

#### `nextBusinessDay(date [, holidays])`

Returns the next business day after the given date.  
`holidays` is an optional array of date strings.

```chariot
nextBusinessDay("2025-06-28") // "2025-06-30T00:00:00Z" (Monday)
```

#### `endOfMonth(date)`

Returns the last day of the month for the given date.

```chariot
endOfMonth("2025-06-15") // "2025-06-30T00:00:00Z"
```

#### `isEndOfMonth(date)`

Returns `true` if the date is the last day of the month.

```chariot
isEndOfMonth("2025-06-30") // true
isEndOfMonth("2025-06-29") // false
```

#### `dateSchedule(start, n, interval [, options])`

Generates a schedule of `n` dates starting from `start`, spaced by the given interval.  
`interval` can be `"day"`, `"week"`, `"month"`, or `"year"`.  
`options` is an optional map with keys:
- `businessDays` (bool): align to business days
- `endOfMonth` (bool): align to end of month
- `holidays` (array): list of holiday dates

```chariot
dateSchedule("2025-01-01", 6, "month") 
// ["2025-01-01T00:00:00Z", "2025-02-01T00:00:00Z", ..., "2025-06-01T00:00:00Z"]

dateSchedule("2025-01-31", 3, "month", map("endOfMonth", true))
// ["2025-01-31T00:00:00Z", "2025-02-28T00:00:00Z", "2025-03-31T00:00:00Z"]
```

#### `parseDate(str, [format])`

Parses a date string and returns the UNIX timestamp (seconds since epoch).  
If `format` is provided, parses using the custom pattern.

```chariot
parseDate("2025-06-27T15:04:05Z") // 1751027045
parseDate("06/27/2025", "MM/DD/YYYY") // 1750982400
```

---

### Notes

- All date functions accept and return date strings in ISO 8601/RFC3339 format unless otherwise specified.
- `parseDate` is flexible and accepts many common date formats.
- Financial conventions for `dayCount` and `yearFraction` are industry standard.
- `formatDate` and `localTime` use a pattern similar to Excel/SQL, not Go's native formatting.
- Use `setTimezone` to change the default timezone for conversions.

---