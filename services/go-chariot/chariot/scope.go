package chariot

import (
	"strconv"
	"time"
)

// Type codes for variable typing
const (
	// Primitive types
	TypeNumber  = "N" // Numeric value
	TypeString  = "S" // String value
	TypeBoolean = "L" // Logical/Boolean value
	TypeDate    = "D" // Date value

	// Complex types
	TypeArray      = "A" // Array value
	TypeExpression = "E" // Expression value
	TypeXML        = "X" // XML fragment
	TypeJSON       = "J" // JSON value
	TypeMap        = "M" // Map value
	TypeTree       = "T" // Tree structure
	TypeFunction   = "F" // Function reference
	TypeObject     = "O" // HostObject reference
	// Agent/Plan types
	TypePlan = "P" // Plan value (BDI plan)

	// Special types
	TypeVariableExpr = "V" // Variable expression
)

type ScopeEntry struct {
	Value    Value
	TypeCode string
	IsTyped  bool
}

// Scope represents a variable scope with parent hierarchy
type Scope struct {
	vars   map[string]ScopeEntry // Variables in this scope
	parent *Scope
}

// NewScope creates a new variable scope with optional parent
func NewScope(parent *Scope) *Scope {
	return &Scope{
		vars:   make(map[string]ScopeEntry),
		parent: parent,
	}
}

// Delete removes a variable from the current scope
func (s *Scope) Delete(name string) bool {
	if _, ok := s.vars[name]; ok {
		delete(s.vars, name)
		return true
	}
	return false
}

// DeleteGlobal removes a variable from the global scope
func (s *Scope) DeleteGlobal(name string) bool {
	if s.parent != nil {
		if _, ok := s.parent.vars[name]; ok {
			delete(s.parent.vars, name)
			return true
		}
	}
	return false
}

// Set sets a variable in the current scope
func (s *Scope) Set(name string, value Value) {
	// Check if variable exists and has type constraint
	if entry, exists := s.vars[name]; exists && entry.IsTyped {
		// Don't modify type information for existing typed variables
		s.vars[name] = ScopeEntry{
			Value:    value,
			TypeCode: entry.TypeCode,
			IsTyped:  true,
		}
	} else {
		// Untyped variable - store without type constraint
		s.vars[name] = ScopeEntry{
			Value:    value,
			TypeCode: TypeVariableExpr, // Default type for untyped variables
			IsTyped:  false,
		}
	}
}

func (s *Scope) SetWithType(name string, value Value, typeCode string) {
	s.vars[name] = ScopeEntry{
		Value:    value,
		TypeCode: typeCode,
		IsTyped:  (typeCode != TypeVariableExpr),
	}
}

// Get retrieves a variable from this scope or parent scopes
func (s *Scope) Get(name string) (Value, bool) {
	// Check current scope
	if entry, ok := s.vars[name]; ok {
		return entry.Value, true // Return just the VALUE, not the whole entry
	}

	// Check parent scope if available
	if s.parent != nil {
		return s.parent.Get(name)
	}

	return nil, false
}

// GetEntry retrieves a variable entry with type information from this scope or parent scopes
func (s *Scope) GetEntry(name string) (ScopeEntry, bool) {
	// Check current scope
	if entry, exists := s.vars[name]; exists {
		return entry, true
	}

	// Check parent scope if available
	if s.parent != nil {
		return s.parent.GetEntry(name)
	}

	return ScopeEntry{}, false
}

// IsTypeCompatible checks if a value is compatible with the specified type code
func IsTypeCompatible(value Value, typeCode string) bool {

	switch typeCode {
	case TypeNumber:
		_, ok := value.(Number)
		return ok

	case TypeString:
		_, ok := value.(Str)
		return ok

	case TypeBoolean:
		_, ok := value.(Bool)
		return ok

	case TypeDate:
		if str, ok := value.(Str); ok {
			// Simple date validation - enhance as needed
			return IsDateString(string(str))
		}
		return false

	case TypeArray:
		_, ok := value.(*ArrayValue)
		return ok

	case TypeXML:
		_, ok := value.(TreeNode)
		return ok

	case TypeJSON:
		// Could be various types
		return true

	case TypeObject:
		// Host object check
		_, ok := value.(*HostObjectValue)
		return ok

	case TypeVariableExpr:
		return true

	default:
		// Unknown type - assume compatible for flexibility
		return true
	}
}

// IsDateString validates both simple YYYY-MM-DD dates and RFC3339 datetime strings
func IsDateString(s string) bool {
	// Empty string is not a date
	if len(s) == 0 {
		return false
	}

	// Case 1: Simple YYYY-MM-DD format
	if len(s) == 10 {
		// Basic check for format YYYY-MM-DD
		for i, c := range s {
			switch i {
			case 4, 7:
				if c != '-' {
					return false
				}
			default:
				if c < '0' || c > '9' {
					return false
				}
			}
		}

		// Validate month and day values
		year, _ := strconv.Atoi(s[0:4])
		month, _ := strconv.Atoi(s[5:7])
		day, _ := strconv.Atoi(s[8:10])

		return validateYearMonthDay(year, month, day)
	}

	// Case 2: RFC3339 datetime format
	// First, check if it looks like an RFC3339 string
	if len(s) >= 20 && s[4] == '-' && s[7] == '-' && (s[10] == 'T' || s[10] == ' ') {
		// Try parsing with Go's time package
		_, err := time.Parse(time.RFC3339, s)
		if err == nil {
			return true
		}

		// Also try with space instead of 'T' separator
		if s[10] == ' ' {
			modified := s[:10] + "T" + s[11:]
			_, err = time.Parse(time.RFC3339, modified)
			return err == nil
		}
	}

	return false
}

// validateYearMonthDay checks if the date values are valid
func validateYearMonthDay(year, month, day int) bool {
	// Basic validation
	if year < 1 || month < 1 || month > 12 || day < 1 || day > 31 {
		return false
	}

	// Check days in month with leap year handling
	daysInMonth := []int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

	// Adjust February for leap years
	if month == 2 && isLeapYear(year) {
		daysInMonth[2] = 29
	}

	return day <= daysInMonth[month]
}

// isLeapYear checks if the given year is a leap year
func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func isValidTypeCode(typeCode string) bool {
	switch typeCode {
	case TypeNumber, TypeString, TypeBoolean, TypeDate,
		TypeArray, TypeExpression, TypeXML, TypeJSON,
		TypeMap, TypeTree, TypeFunction, TypeObject,
		TypePlan,
		TypeVariableExpr:
		return true
	default:
		return false
	}
}
