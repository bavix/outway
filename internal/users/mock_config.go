package users

import (
	"github.com/bavix/outway/internal/config"
)

// MockConfig is a mock implementation of config.Config for testing.
type MockConfig struct {
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
	return m.Users
}

func (m *MockConfig) SetUsers(users []config.UserConfig) {
	m.Users = users
}
