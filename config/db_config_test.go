package config

import (
	"os"
	"testing"
)

func TestLoadDBConfig_Defaults(t *testing.T) {
	t.Cleanup(func() {
		_ = os.Unsetenv("DB_HOST")
		_ = os.Unsetenv("DB_PORT")
	})
	c := LoadDBConfig()
	if c.Host != "localhost" || c.Port != "5440" || c.User != "admin" {
		t.Fatalf("defaults: %+v", c)
	}
}

func TestLoadDBConfig_Env(t *testing.T) {
	t.Setenv("DB_HOST", "db.example")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_NAME", "appdb")
	c := LoadDBConfig()
	if c.Host != "db.example" || c.Port != "5433" || c.DBName != "appdb" {
		t.Fatalf("env: %+v", c)
	}
}

