package internal

import "testing"

func TestConfigValidateAppliesDefaults(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "db.example.local",
			Username: "sa",
			Password: "secret",
			Port:     1433,
		},
		Email: EmailConfig{
			Username: "robot@example.local",
			Password: "mail-secret",
			Hostname: "smtp.example.local",
			Port:     587,
		},
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	if cfg.Report.Subject == "" {
		t.Fatal("expected default report subject")
	}
	if cfg.Report.CompanyName == "" {
		t.Fatal("expected default company name")
	}
	if cfg.Report.Signature == "" {
		t.Fatal("expected default signature")
	}
}

func TestConfigValidateMissingFields(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
