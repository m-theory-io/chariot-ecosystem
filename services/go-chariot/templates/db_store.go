package templates

import (
	"database/sql"
	"encoding/json"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// DBTemplateStore implements TemplateStore using a database
type DBTemplateStore struct {
	db *sql.DB
}

// NewDBTemplateStore creates a new database-backed template store
func NewDBTemplateStore(db *sql.DB) *DBTemplateStore {
	return &DBTemplateStore{db: db}
}

// GetTemplate retrieves a template by ID
func (s *DBTemplateStore) GetTemplate(id string) (*chariot.Template, error) {
	var tmpl chariot.Template
	var parametersJSON string

	query := `SELECT id, name, description, source, author, created, modified, 
              version, parameters_json, hash FROM templates WHERE id = $1`

	err := s.db.QueryRow(query, id).Scan(
		&tmpl.ID,
		&tmpl.Name,
		&tmpl.Description,
		&tmpl.Source,
		&tmpl.Author,
		&tmpl.Created,
		&tmpl.Modified,
		&tmpl.Version,
		&parametersJSON,
		&tmpl.Hash,
	)

	if err != nil {
		return nil, err
	}

	// Parse parameters JSON
	if err := json.Unmarshal([]byte(parametersJSON), &tmpl.Parameters); err != nil {
		return nil, err
	}

	return &tmpl, nil
}

// More methods for SaveTemplate, ListTemplates, DeleteTemplate...
