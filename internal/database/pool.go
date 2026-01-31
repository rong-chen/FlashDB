package database

import (
	"fmt"
	"sync"

	"flash-db/internal/database/mysql"
)

// Pool manages database connections
type Pool struct {
	mu      sync.RWMutex
	drivers map[string]Driver
}

// NewPool creates a new connection pool
func NewPool() *Pool {
	return &Pool{
		drivers: make(map[string]Driver),
	}
}

// Connect creates a new connection and adds it to the pool
func (p *Pool) Connect(id string, config DriverConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Close existing connection if any
	if existing, ok := p.drivers[id]; ok {
		existing.Close()
		delete(p.drivers, id)
	}

	// Create driver based on type
	var driver Driver
	switch config.Type {
	case "mysql":
		driver = mysql.New(mysql.Config{
			Host:     config.Host,
			Port:     config.Port,
			Username: config.Username,
			Password: config.Password,
			Database: config.Database,
			SSLMode:  config.SSLMode,
		})
	default:
		return fmt.Errorf("unsupported database type: %s", config.Type)
	}

	// Connect
	if err := driver.Connect(); err != nil {
		return err
	}

	p.drivers[id] = driver
	return nil
}

// Get returns a driver from the pool
func (p *Pool) Get(id string) (Driver, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	driver, ok := p.drivers[id]
	return driver, ok
}

// Disconnect closes and removes a connection from the pool
func (p *Pool) Disconnect(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if driver, ok := p.drivers[id]; ok {
		err := driver.Close()
		delete(p.drivers, id)
		return err
	}
	return nil
}

// IsConnected checks if a connection exists and is alive
func (p *Pool) IsConnected(id string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if driver, ok := p.drivers[id]; ok {
		return driver.Ping() == nil
	}
	return false
}

// Close closes all connections in the pool
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, driver := range p.drivers {
		driver.Close()
		delete(p.drivers, id)
	}
}

// List returns list of connected IDs
func (p *Pool) List() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ids := make([]string, 0, len(p.drivers))
	for id := range p.drivers {
		ids = append(ids, id)
	}
	return ids
}
