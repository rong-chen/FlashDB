package models

import "time"

// Connection represents a database connection configuration
type Connection struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // mysql
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Username    string    `json:"username"`
	Password    string    `json:"password"` // encrypted
	Database    string    `json:"database"`
	SSLMode     string    `json:"sslMode"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ConnectionInfo is the public view without password
type ConnectionInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Username  string    `json:"username"`
	Database  string    `json:"database"`
	SSLMode   string    `json:"sslMode"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ConnectionRequest is used for creating/updating connections
type ConnectionRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"sslMode"`
}

// ToInfo converts Connection to ConnectionInfo (without password)
func (c *Connection) ToInfo() ConnectionInfo {
	return ConnectionInfo{
		ID:        c.ID,
		Name:      c.Name,
		Type:      c.Type,
		Host:      c.Host,
		Port:      c.Port,
		Username:  c.Username,
		Database:  c.Database,
		SSLMode:   c.SSLMode,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
