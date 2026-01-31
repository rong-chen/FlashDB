package models

// DatabaseInfo represents a database
type DatabaseInfo struct {
	Name string `json:"name"`
}

// TableInfo represents a table in a database
type TableInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // TABLE, VIEW
	Rows    int64  `json:"rows"`
	Comment string `json:"comment"`
}

// ColumnInfo represents a column in a table
type ColumnInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable"`
	Key          string `json:"key"` // PRI, UNI, MUL
	Default      string `json:"default"`
	Extra        string `json:"extra"` // auto_increment, etc.
	Comment      string `json:"comment"`
	CharacterSet string `json:"characterSet"`
	Collation    string `json:"collation"`
}

// IndexInfo represents an index on a table
type IndexInfo struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns"`
	Unique     bool     `json:"unique"`
	Primary    bool     `json:"primary"`
	Type       string   `json:"type"` // BTREE, HASH, FULLTEXT
}

// SchemaObject represents a generic schema object (for tree)
type SchemaObject struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"` // database, table, view, column, index
	Children []SchemaObject `json:"children,omitempty"`
	Data     interface{}    `json:"data,omitempty"`
}
