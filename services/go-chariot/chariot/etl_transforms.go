package chariot

// ETL Transform Registry
type ETLTransformRegistry struct {
	transforms map[string]*ETLTransform
}

type ETLTransform struct {
	Name        string
	Description string
	DataType    string   // Expected output type
	Program     []string // Chariot validation/transformation program
	Examples    []string // Example inputs/outputs
	Category    string   // "validation", "formatting", "conversion", etc.
}

func NewETLTransformRegistry() *ETLTransformRegistry {
	registry := &ETLTransformRegistry{
		transforms: make(map[string]*ETLTransform),
	}

	// Register common transforms
	registry.registerBuiltinTransforms()

	return registry
}

func (r *ETLTransformRegistry) Register(transform *ETLTransform) {
	r.transforms[transform.Name] = transform
}

func (r *ETLTransformRegistry) Get(name string) (*ETLTransform, bool) {
	transform, exists := r.transforms[name]
	return transform, exists
}

func (r *ETLTransformRegistry) List() []string {
	names := make([]string, 0, len(r.transforms))
	for name := range r.transforms {
		names = append(names, name)
	}
	return names
}

func (r *ETLTransformRegistry) registerBuiltinTransforms() {
	// SSN Validation and Formatting
	r.Register(&ETLTransform{
		Name:        "ssn",
		Description: "Validates and formats Social Security Numbers",
		DataType:    "VARCHAR",
		Category:    "validation",
		Program: []string{
			`# Clean input - remove all non-digits`,
			`setq(cleaned, regexReplace(sourceValue, '[^0-9]', ''))`,
			``,
			`# Validate length`,
			`if(unequal(length(cleaned), 9),`,
			`    error(concat('Invalid SSN length: ', sourceValue)),`,
			`    # Format as XXX-XX-XXXX`,
			`    concat(`,
			`        substring(cleaned, 0, 3), '-',`,
			`        substring(cleaned, 3, 2), '-',`,
			`        substring(cleaned, 5, 4)`,
			`    )`,
			`)`,
		},
		Examples: []string{
			"123456789 → 123-45-6789",
			"123-45-6789 → 123-45-6789",
			"123.45.6789 → 123-45-6789",
			"12345 → ERROR: Invalid SSN length",
		},
	})

	// Email Validation and Normalization
	r.Register(&ETLTransform{
		Name:        "email",
		Description: "Validates and normalizes email addresses",
		DataType:    "VARCHAR",
		Category:    "validation",
		Program: []string{
			`# Trim and convert to lowercase`,
			`setq(cleaned, toLowerCase(trim(sourceValue)))`,
			``,
			`# Basic email validation`,
			`if(not(contains(cleaned, '@')),`,
			`    error(concat('Invalid email format: ', sourceValue)),`,
			`    if(not(regexMatch(cleaned, '^[^@]+@[^@]+\\.[^@]+$')),`,
			`        error(concat('Invalid email format: ', sourceValue)),`,
			`        cleaned`,
			`    )`,
			`)`,
		},
		Examples: []string{
			"John.Doe@EXAMPLE.COM → john.doe@example.com",
			"  test@test.com  → test@test.com",
			"invalid-email → ERROR: Invalid email format",
		},
	})

	// Phone Number Formatting
	r.Register(&ETLTransform{
		Name:        "phone_us",
		Description: "Validates and formats US phone numbers",
		DataType:    "VARCHAR",
		Category:    "formatting",
		Program: []string{
			`# Remove all non-digits`,
			`setq(digits, regexReplace(sourceValue, '[^0-9]', ''))`,
			``,
			`# Handle different lengths`,
			`if(equal(length(digits), 10),`,
			`    # Format as (XXX) XXX-XXXX`,
			`    concat('(', substring(digits, 0, 3), ') ',`,
			`           substring(digits, 3, 3), '-',`,
			`           substring(digits, 6, 4)),`,
			`    if(equal(length(digits), 11),`,
			`        if(equal(substring(digits, 0, 1), '1'),`,
			`            # Remove leading 1 and format`,
			`            concat('(', substring(digits, 1, 3), ') ',`,
			`                   substring(digits, 4, 3), '-',`,
			`                   substring(digits, 7, 4)),`,
			`            error(concat('Invalid phone number: ', sourceValue))`,
			`        ),`,
			`        error(concat('Invalid phone number length: ', sourceValue))`,
			`    )`,
			`)`,
		},
		Examples: []string{
			"1234567890 → (123) 456-7890",
			"11234567890 → (123) 456-7890",
			"(123) 456-7890 → (123) 456-7890",
			"123-456-7890 → (123) 456-7890",
		},
	})

	// Currency/Money Formatting
	r.Register(&ETLTransform{
		Name:        "currency_usd",
		Description: "Converts currency strings to decimal cents",
		DataType:    "INT",
		Category:    "conversion",
		Program: []string{
			`# Remove currency symbols and spaces`,
			`setq(cleaned, regexReplace(sourceValue, '[$,\\s]', ''))`,
			``,
			`# Convert to float and multiply by 100 for cents`,
			`if(isNumeric(cleaned),`,
			`    round(multiply(parseFloat(cleaned), 100)),`,
			`    error(concat('Invalid currency amount: ', sourceValue))`,
			`)`,
		},
		Examples: []string{
			"$123.45 → 12345",
			"1,234.56 → 123456",
			"$ 99.99 → 9999",
			"invalid → ERROR: Invalid currency amount",
		},
	})

	// Date Parsing and Formatting
	r.Register(&ETLTransform{
		Name:        "date_mdy",
		Description: "Parses MM/DD/YYYY dates and converts to ISO format",
		DataType:    "DATE",
		Category:    "conversion",
		Program: []string{
			`# Try to parse common date formats`,
			`if(equal(trim(sourceValue), ''),`,
			`    if(required, error('Date is required'), null),`,
			`    if(regexMatch(sourceValue, '^\\d{1,2}/\\d{1,2}/\\d{4}$'),`,
			`        # MM/DD/YYYY format`,
			`        formatDate(parseDate(sourceValue, 'MM/dd/yyyy'), 'yyyy-MM-dd'),`,
			`        if(regexMatch(sourceValue, '^\\d{4}-\\d{2}-\\d{2}$'),`,
			`            # Already in ISO format`,
			`            sourceValue,`,
			`            error(concat('Invalid date format: ', sourceValue))`,
			`        )`,
			`    )`,
			`)`,
		},
		Examples: []string{
			"12/25/2023 → 2023-12-25",
			"1/1/2024 → 2024-01-01",
			"2023-12-25 → 2023-12-25",
			"invalid → ERROR: Invalid date format",
		},
	})

	// Boolean Conversion
	r.Register(&ETLTransform{
		Name:        "boolean",
		Description: "Converts various boolean representations to true/false",
		DataType:    "BOOLEAN",
		Category:    "conversion",
		Program: []string{
			`setq(lower, toLowerCase(trim(sourceValue)))`,
			``,
			`if(or(equal(lower, 'true'), equal(lower, 'yes'), equal(lower, 'y'),`,
			`      equal(lower, '1'), equal(lower, 'on'), equal(lower, 'active')),`,
			`    true,`,
			`    if(or(equal(lower, 'false'), equal(lower, 'no'), equal(lower, 'n'),`,
			`          equal(lower, '0'), equal(lower, 'off'), equal(lower, 'inactive')),`,
			`        false,`,
			`        error(concat('Invalid boolean value: ', sourceValue))`,
			`    )`,
			`)`,
		},
		Examples: []string{
			"YES → true",
			"No → false",
			"1 → true",
			"0 → false",
			"Active → true",
			"maybe → ERROR: Invalid boolean value",
		},
	})
}
