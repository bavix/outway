package users

import (
	"sync"

	"github.com/bavix/outway/internal/config"
)

// MockConfig is a mock implementation of config.Config for testing.
type MockConfig struct {
	mu        sync.RWMutex
	Users     []config.UserConfig
	SaveError error
}

func (m *MockConfig) Save() error {
	return m.SaveError
}

func (m *MockConfig) Load() error {
	return nil
}

func (m *MockConfig) GetUsers() []config.UserConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.Users
}

func (m *MockConfig) SetUsers(users []config.UserConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Users = users
}
