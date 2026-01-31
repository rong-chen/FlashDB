package models

import "time"

// QueryRequest represents a SQL query request
type QueryRequest struct {
	ConnectionID string `json:"connectionId"`
	Database     string `json:"database"`
	SQL          string `json:"sql"`
	Limit        int    `json:"limit"`
	Offset       int    `json:"offset"`
}

// QueryResult represents the result of a SQL query
type QueryResult struct {
	Columns   []ColumnMeta    `json:"columns"`
	Rows      [][]interface{} `json:"rows"`
	RowCount  int             `json:"rowCount"`
	ExecTime  int64           `json:"execTime"` // milliseconds
	Affected  int64           `json:"affected"` // for UPDATE/DELETE/INSERT
	LastID    int64           `json:"lastId"`   // for INSERT
	Error     string          `json:"error,omitempty"`
	Truncated bool            `json:"truncated"` // if results were limited
}

// ColumnMeta represents metadata about a result column
type ColumnMeta struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// QueryHistory represents a historical query
type QueryHistory struct {
	ID           string    `json:"id"`
	ConnectionID string    `json:"connectionId"`
	Database     string    `json:"database"`
	SQL          string    `json:"sql"`
	ExecTime     int64     `json:"execTime"`
	RowCount     int       `json:"rowCount"`
	Error        string    `json:"error,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// EditorTab represents an editor tab state
type EditorTab struct {
	ID           string `json:"id"`
	ConnectionID string `json:"connectionId"`
	Database     string `json:"database"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	IsDirty      bool   `json:"isDirty"`
}
