package internal

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Port     int    `yaml:"port"`
}

type EmailConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Hostname string `yaml:"hostname"`
	Port     int    `yaml:"port"`
}

type ReportConfig struct {
	Subject     string `yaml:"subject"`
	CompanyName string `yaml:"company_name"`
	Signature   string `yaml:"signature"`
}

// Config stores runtime configuration loaded from YAML.
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Email    EmailConfig    `yaml:"email"`
	Report   ReportConfig   `yaml:"report"`
}

// NewConfig loads config from disk, applies defaults and validates required values.
func NewConfig(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (cfg *Config) applyDefaults() {
	if strings.TrimSpace(cfg.Report.Subject) == "" {
		cfg.Report.Subject = "Raport zaleglych faktur"
	}
	if strings.TrimSpace(cfg.Report.CompanyName) == "" {
		cfg.Report.CompanyName = "Nasza Era Sp.z o.o."
	}
	if strings.TrimSpace(cfg.Report.Signature) == "" {
		cfg.Report.Signature = "Janusz"
	}
}

// Validate checks whether required config values are present.
func (cfg *Config) Validate() error {
	missing := make([]string, 0, 6)

	if strings.TrimSpace(cfg.Database.Host) == "" {
		missing = append(missing, "database.host")
	}
	if strings.TrimSpace(cfg.Database.Username) == "" {
		missing = append(missing, "database.username")
	}
	if strings.TrimSpace(cfg.Database.Password) == "" {
		missing = append(missing, "database.password")
	}
	if strings.TrimSpace(cfg.Email.Username) == "" {
		missing = append(missing, "email.username")
	}
	if strings.TrimSpace(cfg.Email.Password) == "" {
		missing = append(missing, "email.password")
	}
	if strings.TrimSpace(cfg.Email.Hostname) == "" {
		missing = append(missing, "email.hostname")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing config values: %s", strings.Join(missing, ", "))
	}
	if cfg.Database.Port <= 0 {
		return fmt.Errorf("database.port must be greater than zero")
	}
	if cfg.Email.Port <= 0 {
		return fmt.Errorf("email.port must be greater than zero")
	}

	return nil
}

// ConnectionString returns the DSN for SQL Server.
func (db DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d",
		db.Host,
		db.Username,
		db.Password,
		db.Port)
}
