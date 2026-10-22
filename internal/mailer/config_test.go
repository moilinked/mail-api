package mailer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigSelectsProvider(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantHost string
		wantPort int
		wantEnc  string
		wantUser string
	}{
		{
			name:     "mail",
			body:     "provider: mail\nmail:\n  host: smtp.feishu.cn\n  port: 587\n  username: info@waterdropfilter.com\n  password: secret\n  encryption: starttls\n",
			wantHost: "smtp.feishu.cn",
			wantPort: 587,
			wantEnc:  encryptionSTARTTLS,
			wantUser: "info@waterdropfilter.com",
		},
		{
			name:     "aliyun",
			body:     "provider: aliyun\naliyun:\n  host: smtpdm.aliyun.com\n  port: 465\n  username: notify@example.com\n  password: secret\n  encryption: ssl\n",
			wantHost: "smtpdm.aliyun.com",
			wantPort: 465,
			wantEnc:  encryptionSSL,
			wantUser: "notify@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfig(t, tt.body)
			cfg, err := LoadConfig(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if cfg.Provider != tt.name || cfg.Host != tt.wantHost || cfg.Port != tt.wantPort || cfg.Encryption != tt.wantEnc || cfg.Username != tt.wantUser {
				t.Fatalf("config = %+v", cfg)
			}
		})
	}
}

func TestLoadConfigRejectsUnknownProvider(t *testing.T) {
	path := writeConfig(t, "provider: sendgrid\n")
	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "unsupported mail provider") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadConfigRequiresAliyunCredentials(t *testing.T) {
	path := writeConfig(t, "provider: aliyun\naliyun:\n  username: \"\"\n  password: \"\"\n")
	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "aliyun smtp config requires host, port, username, password, encryption") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadRepoConfigUsesMail(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Provider != providerMail || cfg.Host != "smtp.feishu.cn" || cfg.Port != 587 || cfg.Encryption != encryptionSTARTTLS {
		t.Fatalf("provider = %s host = %s port = %d encryption = %s", cfg.Provider, cfg.Host, cfg.Port, cfg.Encryption)
	}
	if cfg.Username != "info@waterdropfilter.com" || cfg.Password == "" {
		t.Fatal("expected mail credentials from config")
	}
}

func TestNewAcceptsBothEncryptions(t *testing.T) {
	tests := []Config{
		{Host: "smtp.feishu.cn", Port: 587, Username: "info@waterdropfilter.com", Password: "secret", Encryption: encryptionSTARTTLS},
		{Host: "smtpdm.aliyun.com", Port: 465, Username: "notify@example.com", Password: "secret", Encryption: encryptionSSL},
	}
	for _, cfg := range tests {
		if _, err := New(cfg); err != nil {
			t.Fatalf("new %s: %v", cfg.Encryption, err)
		}
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
