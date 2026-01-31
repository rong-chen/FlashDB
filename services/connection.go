package services

import (
	"context"
	"fmt"

	"flash-db/internal/database"
	"flash-db/internal/storage"
	"flash-db/models"
)

// ConnectionService handles database connection management
type ConnectionService struct {
	ctx     context.Context
	storage *storage.Storage
	pool    *database.Pool
}

// NewConnectionService creates a new ConnectionService
func NewConnectionService(storage *storage.Storage, pool *database.Pool) *ConnectionService {
	return &ConnectionService{
		storage: storage,
		pool:    pool,
	}
}

// SetContext sets the Wails context
func (s *ConnectionService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// Response wraps API responses
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ListConnections returns all saved connections
func (s *ConnectionService) ListConnections() Response {
	connections, err := s.storage.ListConnections()
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	// Add connection status
	type ConnectionWithStatus struct {
		models.ConnectionInfo
		Connected bool `json:"connected"`
	}

	result := make([]ConnectionWithStatus, len(connections))
	for i, conn := range connections {
		result[i] = ConnectionWithStatus{
			ConnectionInfo: conn,
			Connected:      s.pool.IsConnected(conn.ID),
		}
	}

	return Response{Success: true, Data: result}
}

// CreateConnection creates a new connection configuration
func (s *ConnectionService) CreateConnection(req models.ConnectionRequest) Response {
	conn, err := s.storage.CreateConnection(req)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}
	return Response{Success: true, Data: conn.ToInfo()}
}

// UpdateConnection updates an existing connection
func (s *ConnectionService) UpdateConnection(id string, req models.ConnectionRequest) Response {
	conn, err := s.storage.UpdateConnection(id, req)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}
	return Response{Success: true, Data: conn.ToInfo()}
}

// DeleteConnection deletes a connection
func (s *ConnectionService) DeleteConnection(id string) Response {
	// Disconnect first if connected
	s.pool.Disconnect(id)

	if err := s.storage.DeleteConnection(id); err != nil {
		return Response{Success: false, Error: err.Error()}
	}
	return Response{Success: true}
}

// Connect establishes a database connection
func (s *ConnectionService) Connect(id string) Response {
	conn, err := s.storage.GetConnectionWithPassword(id)
	if err != nil {
		return Response{Success: false, Error: fmt.Sprintf("Failed to get connection: %v", err)}
	}

	config := database.DriverConfig{
		Type:     conn.Type,
		Host:     conn.Host,
		Port:     conn.Port,
		Username: conn.Username,
		Password: conn.Password,
		Database: conn.Database,
		SSLMode:  conn.SSLMode,
	}

	if err := s.pool.Connect(id, config); err != nil {
		return Response{Success: false, Error: fmt.Sprintf("Failed to connect: %v", err)}
	}

	return Response{Success: true, Data: conn.ToInfo()}
}

// Disconnect closes a database connection
func (s *ConnectionService) Disconnect(id string) Response {
	if err := s.pool.Disconnect(id); err != nil {
		return Response{Success: false, Error: err.Error()}
	}
	return Response{Success: true}
}

// TestConnection tests a connection without saving
func (s *ConnectionService) TestConnection(req models.ConnectionRequest) Response {
	config := database.DriverConfig{
		Type:     req.Type,
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Database: req.Database,
		SSLMode:  req.SSLMode,
	}

	// Create a temporary pool for testing
	tempPool := database.NewPool()
	defer tempPool.Close()

	if err := tempPool.Connect("test", config); err != nil {
		return Response{Success: false, Error: fmt.Sprintf("Connection failed: %v", err)}
	}

	return Response{Success: true, Data: "Connection successful"}
}

// IsConnected checks if a connection is active
func (s *ConnectionService) IsConnected(id string) Response {
	connected := s.pool.IsConnected(id)
	return Response{Success: true, Data: connected}
}
