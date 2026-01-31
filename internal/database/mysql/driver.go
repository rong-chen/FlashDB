package mysql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"flash-db/models"

	_ "github.com/go-sql-driver/mysql"
)

// Config contains configuration for MySQL driver
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	SSLMode  string
}

// Driver implements the database.Driver interface for MySQL
type Driver struct {
	config Config
	db     *sql.DB
	connID string
}

// New creates a new MySQL driver
func New(config Config) *Driver {
	return &Driver{
		config: config,
		connID: fmt.Sprintf("%s@%s:%d", config.Username, config.Host, config.Port),
	}
}

// Connect establishes a connection to MySQL
func (d *Driver) Connect() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=10s",
		d.config.Username,
		d.config.Password,
		d.config.Host,
		d.config.Port,
		d.config.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Minute * 5)

	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}

	d.db = db
	return nil
}

// Close closes the database connection
func (d *Driver) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Ping tests the database connection
func (d *Driver) Ping() error {
	if d.db == nil {
		return fmt.Errorf("not connected")
	}
	return d.db.Ping()
}

// GetDatabases returns list of databases
func (d *Driver) GetDatabases() ([]models.DatabaseInfo, error) {
	rows, err := d.db.Query("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []models.DatabaseInfo
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		databases = append(databases, models.DatabaseInfo{Name: name})
	}
	return databases, nil
}

// GetTables returns list of tables in a database
func (d *Driver) GetTables(database string) ([]models.TableInfo, error) {
	query := `
		SELECT
			TABLE_NAME,
			TABLE_TYPE,
			IFNULL(TABLE_ROWS, 0),
			IFNULL(TABLE_COMMENT, '')
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME
	`
	rows, err := d.db.Query(query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []models.TableInfo
	for rows.Next() {
		var t models.TableInfo
		var tableType string
		if err := rows.Scan(&t.Name, &tableType, &t.Rows, &t.Comment); err != nil {
			return nil, err
		}
		if tableType == "VIEW" {
			t.Type = "VIEW"
		} else {
			t.Type = "TABLE"
		}
		tables = append(tables, t)
	}
	return tables, nil
}

// GetColumns returns columns for a table
func (d *Driver) GetColumns(database, table string) ([]models.ColumnInfo, error) {
	query := `
		SELECT
			COLUMN_NAME,
			COLUMN_TYPE,
			IS_NULLABLE,
			IFNULL(COLUMN_KEY, ''),
			IFNULL(COLUMN_DEFAULT, ''),
			IFNULL(EXTRA, ''),
			IFNULL(COLUMN_COMMENT, ''),
			IFNULL(CHARACTER_SET_NAME, ''),
			IFNULL(COLLATION_NAME, '')
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`
	rows, err := d.db.Query(query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []models.ColumnInfo
	for rows.Next() {
		var c models.ColumnInfo
		var nullable string
		var defaultVal sql.NullString
		if err := rows.Scan(&c.Name, &c.Type, &nullable, &c.Key, &defaultVal, &c.Extra, &c.Comment, &c.CharacterSet, &c.Collation); err != nil {
			return nil, err
		}
		c.Nullable = nullable == "YES"
		if defaultVal.Valid {
			c.Default = defaultVal.String
		}
		columns = append(columns, c)
	}
	return columns, nil
}

// GetIndexes returns indexes for a table
func (d *Driver) GetIndexes(database, table string) ([]models.IndexInfo, error) {
	query := `
		SELECT
			INDEX_NAME,
			COLUMN_NAME,
			NON_UNIQUE,
			INDEX_TYPE
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`
	rows, err := d.db.Query(query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*models.IndexInfo)
	var indexOrder []string

	for rows.Next() {
		var indexName, columnName, indexType string
		var nonUnique int
		if err := rows.Scan(&indexName, &columnName, &nonUnique, &indexType); err != nil {
			return nil, err
		}

		if _, exists := indexMap[indexName]; !exists {
			indexMap[indexName] = &models.IndexInfo{
				Name:    indexName,
				Columns: []string{},
				Unique:  nonUnique == 0,
				Primary: indexName == "PRIMARY",
				Type:    indexType,
			}
			indexOrder = append(indexOrder, indexName)
		}
		indexMap[indexName].Columns = append(indexMap[indexName].Columns, columnName)
	}

	var indexes []models.IndexInfo
	for _, name := range indexOrder {
		indexes = append(indexes, *indexMap[name])
	}
	return indexes, nil
}

// Execute executes a query and returns results
func (d *Driver) Execute(database, sqlQuery string, limit, offset int) (*models.QueryResult, error) {
	start := time.Now()
	result := &models.QueryResult{}

	// Switch database if specified
	if database != "" {
		if _, err := d.db.Exec("USE " + database); err != nil {
			result.Error = err.Error()
			return result, nil
		}
	}

	// Determine if it's a SELECT query
	trimmedSQL := strings.TrimSpace(strings.ToUpper(sqlQuery))
	isSelect := strings.HasPrefix(trimmedSQL, "SELECT") ||
		strings.HasPrefix(trimmedSQL, "SHOW") ||
		strings.HasPrefix(trimmedSQL, "DESCRIBE") ||
		strings.HasPrefix(trimmedSQL, "EXPLAIN")

	if isSelect {
		return d.executeSelect(sqlQuery, limit, offset, start)
	}

	return d.executeNonSelect(sqlQuery, start)
}

func (d *Driver) executeSelect(sqlQuery string, limit, offset int, start time.Time) (*models.QueryResult, error) {
	result := &models.QueryResult{}

	// Add LIMIT if not already present and limit > 0
	if limit > 0 && !strings.Contains(strings.ToUpper(sqlQuery), "LIMIT") {
		// Remove trailing semicolon before adding LIMIT
		trimmedQuery := strings.TrimSpace(sqlQuery)
		trimmedQuery = strings.TrimSuffix(trimmedQuery, ";")
		sqlQuery = fmt.Sprintf("%s LIMIT %d OFFSET %d", trimmedQuery, limit, offset)
	}

	rows, err := d.db.Query(sqlQuery)
	if err != nil {
		result.Error = err.Error()
		result.ExecTime = time.Since(start).Milliseconds()
		return result, nil
	}
	defer rows.Close()

	// Get column info
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		result.Error = err.Error()
		result.ExecTime = time.Since(start).Milliseconds()
		return result, nil
	}

	for _, col := range columnTypes {
		nullable, _ := col.Nullable()
		result.Columns = append(result.Columns, models.ColumnMeta{
			Name:     col.Name(),
			Type:     col.DatabaseTypeName(),
			Nullable: nullable,
		})
	}

	// Scan rows
	colCount := len(columnTypes)
	for rows.Next() {
		values := make([]interface{}, colCount)
		valuePtrs := make([]interface{}, colCount)
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			result.Error = err.Error()
			result.ExecTime = time.Since(start).Milliseconds()
			return result, nil
		}

		// Convert values to JSON-safe types
		row := make([]interface{}, colCount)
		for i, v := range values {
			row[i] = convertValue(v)
		}
		result.Rows = append(result.Rows, row)
	}

	result.RowCount = len(result.Rows)
	result.ExecTime = time.Since(start).Milliseconds()
	return result, nil
}

func (d *Driver) executeNonSelect(sqlQuery string, start time.Time) (*models.QueryResult, error) {
	result := &models.QueryResult{}

	res, err := d.db.Exec(sqlQuery)
	if err != nil {
		result.Error = err.Error()
		result.ExecTime = time.Since(start).Milliseconds()
		return result, nil
	}

	result.Affected, _ = res.RowsAffected()
	result.LastID, _ = res.LastInsertId()
	result.ExecTime = time.Since(start).Milliseconds()
	return result, nil
}

// GetTableDDL returns the CREATE TABLE statement
func (d *Driver) GetTableDDL(database, table string) (string, error) {
	var tableName, createSQL string
	query := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", database, table)
	err := d.db.QueryRow(query).Scan(&tableName, &createSQL)
	if err != nil {
		return "", err
	}
	return createSQL, nil
}

// GetConnectionID returns the connection identifier
func (d *Driver) GetConnectionID() string {
	return d.connID
}

// convertValue converts database values to JSON-safe types
func convertValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []byte:
		return string(val)
	case time.Time:
		return val.Format("2006-01-02 15:04:05")
	default:
		return val
	}
}
