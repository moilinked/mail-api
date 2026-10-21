package mailer

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	providerFeishu     = "feishu"
	providerAliyun     = "aliyun"
	encryptionSTARTTLS = "starttls"
	encryptionSSL      = "ssl"
)

type fileConfig struct {
	Provider string     `yaml:"provider"`
	Feishu   smtpConfig `yaml:"feishu"`
	Aliyun   smtpConfig `yaml:"aliyun"`
}

type smtpConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	Encryption string `yaml:"encryption"`
}

// LoadConfig 读取配置文件，并按 provider 选出飞书或阿里云 SMTP。
func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read mail config: %w", err)
	}

	var file fileConfig
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return Config{}, fmt.Errorf("parse mail config: %w", err)
	}

	provider := strings.ToLower(strings.TrimSpace(file.Provider))
	var selected smtpConfig
	switch provider {
	case providerFeishu:
		selected = file.Feishu
	case providerAliyun:
		selected = file.Aliyun
	default:
		return Config{}, fmt.Errorf("unsupported mail provider %q", file.Provider)
	}
	if err := selected.require(provider); err != nil {
		return Config{}, err
	}

	encryption, err := normalizeEncryption(selected.Encryption)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Provider:   provider,
		Host:       selected.Host,
		Port:       selected.Port,
		Username:   selected.Username,
		Password:   selected.Password,
		Encryption: encryption,
	}, nil
}

func (cfg smtpConfig) require(provider string) error {
	var missing []string
	if strings.TrimSpace(cfg.Host) == "" {
		missing = append(missing, "host")
	}
	if cfg.Port == 0 {
		missing = append(missing, "port")
	}
	if strings.TrimSpace(cfg.Username) == "" {
		missing = append(missing, "username")
	}
	if cfg.Password == "" {
		missing = append(missing, "password")
	}
	if strings.TrimSpace(cfg.Encryption) == "" {
		missing = append(missing, "encryption")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s smtp config requires %s", provider, strings.Join(missing, ", "))
}

func normalizeEncryption(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case encryptionSTARTTLS:
		return encryptionSTARTTLS, nil
	case encryptionSSL:
		return encryptionSSL, nil
	default:
		return "", fmt.Errorf("unsupported smtp encryption %q", value)
	}
}
