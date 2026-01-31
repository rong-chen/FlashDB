package services

import (
	"context"
	"fmt"

	"flash-db/internal/database"
	"flash-db/models"
)

// ExplorerService handles database schema exploration
type ExplorerService struct {
	ctx  context.Context
	pool *database.Pool
}

// NewExplorerService creates a new ExplorerService
func NewExplorerService(pool *database.Pool) *ExplorerService {
	return &ExplorerService{
		pool: pool,
	}
}

// SetContext sets the Wails context
func (s *ExplorerService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// GetDatabases returns list of databases for a connection
func (s *ExplorerService) GetDatabases(connectionID string) Response {
	driver, ok := s.pool.Get(connectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	databases, err := driver.GetDatabases()
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Data: databases}
}

// GetTables returns list of tables in a database
func (s *ExplorerService) GetTables(connectionID, database string) Response {
	driver, ok := s.pool.Get(connectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	tables, err := driver.GetTables(database)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Data: tables}
}

// GetColumns returns columns for a table
func (s *ExplorerService) GetColumns(connectionID, database, table string) Response {
	driver, ok := s.pool.Get(connectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	columns, err := driver.GetColumns(database, table)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Data: columns}
}

// GetIndexes returns indexes for a table
func (s *ExplorerService) GetIndexes(connectionID, database, table string) Response {
	driver, ok := s.pool.Get(connectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	indexes, err := driver.GetIndexes(database, table)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Data: indexes}
}

// GetTableDDL returns the CREATE TABLE statement
func (s *ExplorerService) GetTableDDL(connectionID, database, table string) Response {
	driver, ok := s.pool.Get(connectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	ddl, err := driver.GetTableDDL(database, table)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Data: ddl}
}

// GetSchemaTree returns full schema tree for a connection
func (s *ExplorerService) GetSchemaTree(connectionID string) Response {
	driver, ok := s.pool.Get(connectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	databases, err := driver.GetDatabases()
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	var tree []models.SchemaObject
	for _, db := range databases {
		dbNode := models.SchemaObject{
			Name: db.Name,
			Type: "database",
		}
		tree = append(tree, dbNode)
	}

	return Response{Success: true, Data: tree}
}

// SearchTables searches for tables matching a pattern
func (s *ExplorerService) SearchTables(connectionID, database, pattern string) Response {
	driver, ok := s.pool.Get(connectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	tables, err := driver.GetTables(database)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	// Filter tables by pattern
	var matched []models.TableInfo
	for _, t := range tables {
		if containsIgnoreCase(t.Name, pattern) {
			matched = append(matched, t)
		}
	}

	return Response{Success: true, Data: matched}
}

// TruncateTable truncates a table
func (s *ExplorerService) TruncateTable(connectionID, database, table string) Response {
	driver, ok := s.pool.Get(connectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	sql := fmt.Sprintf("TRUNCATE TABLE `%s`.`%s`", database, table)
	result, err := driver.Execute(database, sql, 0, 0)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}
	if result.Error != "" {
		return Response{Success: false, Error: result.Error}
	}

	return Response{Success: true, Data: "Table truncated"}
}

// DropTable drops a table
func (s *ExplorerService) DropTable(connectionID, database, table string) Response {
	driver, ok := s.pool.Get(connectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	sql := fmt.Sprintf("DROP TABLE `%s`.`%s`", database, table)
	result, err := driver.Execute(database, sql, 0, 0)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}
	if result.Error != "" {
		return Response{Success: false, Error: result.Error}
	}

	return Response{Success: true, Data: "Table dropped"}
}

func containsIgnoreCase(s, substr string) bool {
	return len(substr) == 0 ||
		len(s) >= len(substr) &&
			(s == substr ||
				containsIgnoreCaseImpl(s, substr))
}

func containsIgnoreCaseImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldAt(s, substr, i) {
			return true
		}
	}
	return false
}

func equalFoldAt(s, substr string, start int) bool {
	for i := 0; i < len(substr); i++ {
		c1, c2 := s[start+i], substr[i]
		if c1 != c2 && toLower(c1) != toLower(c2) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// CloneDatabase clones a database structure (and optionally data)
func (s *ExplorerService) CloneDatabase(connectionID, sourceDB, targetDB string, includeData bool) Response {
	driver, ok := s.pool.Get(connectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	// Check if source is a system database
	systemDBs := map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
	}
	if systemDBs[sourceDB] {
		return Response{Success: false, Error: "Cannot clone system database"}
	}

	// Check if target database already exists
	databases, err := driver.GetDatabases()
	if err != nil {
		return Response{Success: false, Error: "Failed to check databases: " + err.Error()}
	}
	for _, db := range databases {
		if db.Name == targetDB {
			return Response{Success: false, Error: fmt.Sprintf("数据库 '%s' 已存在", targetDB)}
		}
	}

	// Create target database
	createSQL := fmt.Sprintf("CREATE DATABASE `%s`", targetDB)
	result, err := driver.Execute("", createSQL, 0, 0)
	if err != nil {
		return Response{Success: false, Error: "Failed to create database: " + err.Error()}
	}
	if result.Error != "" {
		return Response{Success: false, Error: result.Error}
	}

	// Get all tables from source database
	tables, err := driver.GetTables(sourceDB)
	if err != nil {
		return Response{Success: false, Error: "Failed to get tables: " + err.Error()}
	}

	// Clone each table structure
	clonedCount := 0
	for _, table := range tables {
		// Get DDL from source
		ddl, err := driver.GetTableDDL(sourceDB, table.Name)
		if err != nil {
			continue // Skip tables that can't get DDL
		}

		// Create table in target database
		result, err = driver.Execute(targetDB, ddl, 0, 0)
		if err == nil && result.Error == "" {
			clonedCount++

			// Copy data if requested
			if includeData {
				copySQL := fmt.Sprintf("INSERT INTO `%s`.`%s` SELECT * FROM `%s`.`%s`",
					targetDB, table.Name, sourceDB, table.Name)
				driver.Execute(targetDB, copySQL, 0, 0)
			}
		}
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"database":     targetDB,
			"tablesTotal":  len(tables),
			"tablesCopied": clonedCount,
			"includeData":  includeData,
		},
	}
}

// DropDatabase drops a database (requires confirmation)
func (s *ExplorerService) DropDatabase(connectionID, database, confirmName string) Response {
	driver, ok := s.pool.Get(connectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	// Check if it's a system database
	systemDBs := map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
	}
	if systemDBs[database] {
		return Response{Success: false, Error: "Cannot drop system database"}
	}

	// Verify confirmation name matches
	if confirmName != database {
		return Response{Success: false, Error: "数据库名称不匹配，删除取消"}
	}

	// Drop the database
	sql := fmt.Sprintf("DROP DATABASE `%s`", database)
	result, err := driver.Execute("", sql, 0, 0)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}
	if result.Error != "" {
		return Response{Success: false, Error: result.Error}
	}

	return Response{Success: true, Data: fmt.Sprintf("数据库 '%s' 已删除", database)}
}
