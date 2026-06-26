package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultPort              = 11111
	CredentialsLocalVault    = "local-vault"
	Credentials1Password     = "1password"
)

type Config struct {
	Coronagraph CoronagraphConfig `yaml:"coronagraph"`
}

type CoronagraphConfig struct {
	Port        *int      `yaml:"port"`
	Credentials string    `yaml:"credentials"`
	OPSecretRef string    `yaml:"op_secret_ref"`
	TLS         TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	Certificate string `yaml:"certificate"`
	Key         string `yaml:"key"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Coronagraph.Credentials != CredentialsLocalVault && c.Coronagraph.Credentials != Credentials1Password {
		return fmt.Errorf("unsupported credentials backend %q", c.Coronagraph.Credentials)
	}

	if c.Coronagraph.Credentials == Credentials1Password && c.Coronagraph.OPSecretRef == "" {
		return fmt.Errorf("if credential store is 1password, op_secret_ref must be set")
	}

	if err := validateExistingAbsoluteFile("tls.certificate", c.Coronagraph.TLS.Certificate); err != nil {
		return err
	}

	if err := validateExistingAbsoluteFile("tls.key", c.Coronagraph.TLS.Key); err != nil {
		return err
	}

	return nil
}

func validateExistingAbsoluteFile(name, path string) error {
	if path == "" {
		return fmt.Errorf("%s is required", name)
	}

	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s must be an absolute path: %q", name, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s does not exist or is not readable: %w", name, err)
	}

	if info.IsDir() {
		return fmt.Errorf("%s must be a file, got directory: %q", name, path)
	}

	return nil
}

func (c *Config) Port() int {
	if c.Coronagraph.Port == nil {
		return DefaultPort
	}
	return *c.Coronagraph.Port
}

func (c *Config) Credentials() string {
	return c.Coronagraph.Credentials
}

func (c *Config) TLSCertificatePath() string {
	return c.Coronagraph.TLS.Certificate
}

func (c *Config) TLSKeyPath() string {
	return c.Coronagraph.TLS.Key
}

func (c *Config) OPSecretRef() string {
	return c.Coronagraph.OPSecretRef
}
