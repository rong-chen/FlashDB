package models

// UpdateRowRequest represents a request to update a row
type UpdateRowRequest struct {
	ConnectionID string         `json:"connectionId"`
	Database     string         `json:"database"`
	Table        string         `json:"table"`
	PrimaryKey   map[string]any `json:"primaryKey"` // Primary key column(s) and value(s)
	Changes      map[string]any `json:"changes"`    // Column names and new values
}

// InsertRowRequest represents a request to insert a new row
type InsertRowRequest struct {
	ConnectionID string         `json:"connectionId"`
	Database     string         `json:"database"`
	Table        string         `json:"table"`
	Values       map[string]any `json:"values"` // Column names and values
}

// DeleteRowRequest represents a request to delete a row
type DeleteRowRequest struct {
	ConnectionID string         `json:"connectionId"`
	Database     string         `json:"database"`
	Table        string         `json:"table"`
	PrimaryKey   map[string]any `json:"primaryKey"` // Primary key column(s) and value(s)
}

// BatchUpdateRequest represents a request to perform multiple operations
type BatchUpdateRequest struct {
	ConnectionID string             `json:"connectionId"`
	Database     string             `json:"database"`
	Table        string             `json:"table"`
	Updates      []UpdateRowRequest `json:"updates"`
	Inserts      []InsertRowRequest `json:"inserts"`
	Deletes      []DeleteRowRequest `json:"deletes"`
}
