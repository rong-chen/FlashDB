package database

import (
	"flash-db/models"
)

// Driver defines the interface for database drivers
type Driver interface {
	// Connect establishes a connection to the database
	Connect() error
	// Close closes the database connection
	Close() error
	// Ping tests the database connection
	Ping() error
	// GetDatabases returns list of databases
	GetDatabases() ([]models.DatabaseInfo, error)
	// GetTables returns list of tables in a database
	GetTables(database string) ([]models.TableInfo, error)
	// GetColumns returns columns for a table
	GetColumns(database, table string) ([]models.ColumnInfo, error)
	// GetIndexes returns indexes for a table
	GetIndexes(database, table string) ([]models.IndexInfo, error)
	// Execute executes a query and returns results
	Execute(database, sql string, limit, offset int) (*models.QueryResult, error)
	// GetTableDDL returns the CREATE TABLE statement
	GetTableDDL(database, table string) (string, error)
	// GetConnectionID returns the connection identifier
	GetConnectionID() string
}

// DriverConfig contains configuration for creating a driver
type DriverConfig struct {
	Type     string
	Host     string
	Port     int
	Username string
	Password string
	Database string
	SSLMode  string
}
