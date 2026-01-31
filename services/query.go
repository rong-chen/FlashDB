package services

import (
	"context"
	"fmt"
	"strings"

	"flash-db/internal/database"
	"flash-db/models"
)

// QueryService handles SQL query execution
type QueryService struct {
	ctx  context.Context
	pool *database.Pool
}

// NewQueryService creates a new QueryService
func NewQueryService(pool *database.Pool) *QueryService {
	return &QueryService{
		pool: pool,
	}
}

// SetContext sets the Wails context
func (s *QueryService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// Execute executes a SQL query
func (s *QueryService) Execute(req models.QueryRequest) Response {
	driver, ok := s.pool.Get(req.ConnectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	// Default limit
	limit := req.Limit
	if limit <= 0 {
		limit = 1000
	}

	result, err := driver.Execute(req.Database, req.SQL, limit, req.Offset)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	if result.Error != "" {
		return Response{Success: false, Error: result.Error, Data: result}
	}

	return Response{Success: true, Data: result}
}

// ExecuteRaw executes a raw SQL query (for DDL, etc.)
func (s *QueryService) ExecuteRaw(connectionID, database, sql string) Response {
	driver, ok := s.pool.Get(connectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	result, err := driver.Execute(database, sql, 0, 0)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	if result.Error != "" {
		return Response{Success: false, Error: result.Error}
	}

	return Response{Success: true, Data: result}
}

// UpdateRow updates a single row in a table
func (s *QueryService) UpdateRow(req models.UpdateRowRequest) Response {
	driver, ok := s.pool.Get(req.ConnectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	if len(req.PrimaryKey) == 0 {
		return Response{Success: false, Error: "Primary key is required"}
	}

	if len(req.Changes) == 0 {
		return Response{Success: false, Error: "No changes provided"}
	}

	// Build UPDATE SQL
	var setClauses []string
	var whereClauses []string

	for col, val := range req.Changes {
		setClauses = append(setClauses, fmt.Sprintf("`%s` = %s", col, formatValue(val)))
	}

	for col, val := range req.PrimaryKey {
		whereClauses = append(whereClauses, fmt.Sprintf("`%s` = %s", col, formatValue(val)))
	}

	sql := fmt.Sprintf("UPDATE `%s`.`%s` SET %s WHERE %s",
		req.Database, req.Table,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "))

	result, err := driver.Execute(req.Database, sql, 0, 0)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	if result.Error != "" {
		return Response{Success: false, Error: result.Error}
	}

	return Response{Success: true, Data: result}
}

// InsertRow inserts a new row into a table
func (s *QueryService) InsertRow(req models.InsertRowRequest) Response {
	driver, ok := s.pool.Get(req.ConnectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	if len(req.Values) == 0 {
		return Response{Success: false, Error: "No values provided"}
	}

	// Build INSERT SQL
	var columns []string
	var values []string

	for col, val := range req.Values {
		columns = append(columns, fmt.Sprintf("`%s`", col))
		values = append(values, formatValue(val))
	}

	sql := fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) VALUES (%s)",
		req.Database, req.Table,
		strings.Join(columns, ", "),
		strings.Join(values, ", "))

	result, err := driver.Execute(req.Database, sql, 0, 0)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	if result.Error != "" {
		return Response{Success: false, Error: result.Error}
	}

	return Response{Success: true, Data: result}
}

// DeleteRow deletes a row from a table
func (s *QueryService) DeleteRow(req models.DeleteRowRequest) Response {
	driver, ok := s.pool.Get(req.ConnectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	if len(req.PrimaryKey) == 0 {
		return Response{Success: false, Error: "Primary key is required"}
	}

	// Build DELETE SQL
	var whereClauses []string

	for col, val := range req.PrimaryKey {
		whereClauses = append(whereClauses, fmt.Sprintf("`%s` = %s", col, formatValue(val)))
	}

	sql := fmt.Sprintf("DELETE FROM `%s`.`%s` WHERE %s",
		req.Database, req.Table,
		strings.Join(whereClauses, " AND "))

	result, err := driver.Execute(req.Database, sql, 0, 0)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	if result.Error != "" {
		return Response{Success: false, Error: result.Error}
	}

	return Response{Success: true, Data: result}
}

// BatchUpdate performs multiple CRUD operations
func (s *QueryService) BatchUpdate(req models.BatchUpdateRequest) Response {
	driver, ok := s.pool.Get(req.ConnectionID)
	if !ok {
		return Response{Success: false, Error: "Not connected"}
	}

	var affected int64

	// Process updates
	for _, update := range req.Updates {
		update.ConnectionID = req.ConnectionID
		update.Database = req.Database
		update.Table = req.Table
		resp := s.UpdateRow(update)
		if !resp.Success {
			return resp
		}
		if result, ok := resp.Data.(*models.QueryResult); ok {
			affected += result.Affected
		}
	}

	// Process inserts
	for _, insert := range req.Inserts {
		insert.ConnectionID = req.ConnectionID
		insert.Database = req.Database
		insert.Table = req.Table
		resp := s.InsertRow(insert)
		if !resp.Success {
			return resp
		}
		if result, ok := resp.Data.(*models.QueryResult); ok {
			affected += result.Affected
		}
	}

	// Process deletes
	for _, del := range req.Deletes {
		del.ConnectionID = req.ConnectionID
		del.Database = req.Database
		del.Table = req.Table
		resp := s.DeleteRow(del)
		if !resp.Success {
			return resp
		}
		if result, ok := resp.Data.(*models.QueryResult); ok {
			affected += result.Affected
		}
	}

	// Re-fetch table data
	fetchSQL := fmt.Sprintf("SELECT * FROM `%s`.`%s` LIMIT 1000", req.Database, req.Table)
	result, err := driver.Execute(req.Database, fetchSQL, 1000, 0)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	result.Affected = affected
	return Response{Success: true, Data: result}
}

// formatValue formats a value for SQL
func formatValue(v any) string {
	if v == nil {
		return "NULL"
	}

	switch val := v.(type) {
	case string:
		// Escape single quotes
		escaped := strings.ReplaceAll(val, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case bool:
		if val {
			return "1"
		}
		return "0"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%v", val)
	default:
		escaped := strings.ReplaceAll(fmt.Sprintf("%v", val), "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	}
}
