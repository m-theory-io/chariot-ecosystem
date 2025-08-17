package chariot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Template represents a parameterized Chariot script
type Template struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Source      string              `json:"source"` // Original template with placeholders
	Author      string              `json:"author"`
	Created     time.Time           `json:"created"`
	Modified    time.Time           `json:"modified"`
	Version     int                 `json:"version"`
	Parameters  []TemplateParameter `json:"parameters"` // Parameter definitions
	Hash        string              `json:"hash"`       // Security hash
}

// isNumber checks if a value is a number (int or float64)
func (tm *TemplateManager) isNumber(val interface{}) bool {
	switch val.(type) {
	case int, int8, int16, int32, int64, float32, float64:
		return true
	default:
		return false
	}
}

// TemplateParameter defines a parameter for a template
type TemplateParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, number, boolean, date, etc.
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
	Pattern     string      `json:"pattern,omitempty"` // Regex pattern for validation
}

// TemplateStore manages secure storage and retrieval of templates
type TemplateStore interface {
	GetTemplate(id string) (*Template, error)
	SaveTemplate(template *Template) error
	ListTemplates() ([]*Template, error)
	DeleteTemplate(id string) error
}

// TemplateManager handles template operations
type TemplateManager struct {
	store     TemplateStore
	secretKey []byte // For template signing
}

// NewTemplateManager creates a template manager
func NewTemplateManager(store TemplateStore, secretKey string) *TemplateManager {
	return &TemplateManager{
		store:     store,
		secretKey: []byte(secretKey),
	}
}

// RenderTemplate applies parameters to a template
func (tm *TemplateManager) RenderTemplate(templateID string, params map[string]interface{}) (string, error) {
	// Get template
	tmpl, err := tm.store.GetTemplate(templateID)
	if err != nil {
		return "", fmt.Errorf("template not found: %v", err)
	}

	// Verify template integrity
	if !tm.verifyTemplateHash(tmpl) {
		return "", errors.New("template integrity check failed")
	}

	// Validate parameters
	if err := tm.validateParameters(tmpl, params); err != nil {
		return "", err
	}

	// Apply parameters to template
	script := tmpl.Source

	// Replace placeholders with parameter values
	re := regexp.MustCompile(`{{([a-zA-Z0-9_]+)}}`)
	script = re.ReplaceAllStringFunc(script, func(match string) string {
		// Extract parameter name from {{name}}
		paramName := match[2 : len(match)-2]

		if val, exists := params[paramName]; exists {
			return fmt.Sprintf("%v", val)
		}

		// Check if there's a default value
		for _, param := range tmpl.Parameters {
			if param.Name == paramName && param.Default != nil {
				return fmt.Sprintf("%v", param.Default)
			}
		}

		// Keep placeholder if no value or default
		return match
	})

	return script, nil
}

// validateParameters checks that all required parameters are provided and valid
func (tm *TemplateManager) validateParameters(tmpl *Template, params map[string]interface{}) error {
	for _, param := range tmpl.Parameters {
		// Check if required parameter exists
		if param.Required {
			if _, exists := params[param.Name]; !exists {
				return fmt.Errorf("required parameter missing: %s", param.Name)
			}
		}

		// If parameter exists, validate it
		if val, exists := params[param.Name]; exists {
			// Type checking
			switch param.Type {
			case "string":
				if _, ok := val.(string); !ok {
					return fmt.Errorf("parameter %s must be a string", param.Name)
				}

				if _, ok := val.(float64); !ok && !tm.isNumber(val) {
					if param.Pattern != "" {
						strVal := val.(string)
						matched, err := regexp.MatchString(param.Pattern, strVal)
						if err != nil || !matched {
							return fmt.Errorf("parameter %s does not match required pattern", param.Name)
						}
					}
				}
			case "number":
				if _, ok := val.(float64); !ok && !tm.isNumber(val) {
					return fmt.Errorf("parameter %s must be a number", param.Name)
				}

			case "boolean":
				if _, ok := val.(bool); !ok {
					return fmt.Errorf("parameter %s must be a boolean", param.Name)
				}
			}
		}
	}

	return nil
}

// computeTemplateHash generates a hash for template integrity checking
func (tm *TemplateManager) computeTemplateHash(tmpl *Template) string {
	h := hmac.New(sha256.New, tm.secretKey)
	h.Write([]byte(tmpl.ID))
	h.Write([]byte(tmpl.Source))
	h.Write([]byte(fmt.Sprintf("%d", tmpl.Version)))
	return hex.EncodeToString(h.Sum(nil))
}

// verifyTemplateHash checks template integrity
func (tm *TemplateManager) verifyTemplateHash(tmpl *Template) bool {
	expectedHash := tm.computeTemplateHash(tmpl)
	return hmac.Equal([]byte(expectedHash), []byte(tmpl.Hash))
}
