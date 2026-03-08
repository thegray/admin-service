package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/joho/godotenv"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"gopkg.in/yaml.v2"
)

const (
	defaultConfigFile = "config.yaml"
	defaultEnv        = "development"
)

type Config struct {
	Port            string
	Environment     string
	LogLevel        string
	TokenSecret     string
	RateLimitRPS    float64
	RateLimitBurst  int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	UseSecretMgr    bool
}

type fileConfig struct {
	Port             string   `yaml:"port"`
	Environment      string   `yaml:"environment"`
	LogLevel         string   `yaml:"log_level"`
	RateLimitRPS     *float64 `yaml:"rate_limit_rps"`
	RateLimitBurst   *int     `yaml:"rate_limit_burst"`
	ReadTimeoutSecs  *int     `yaml:"read_timeout_seconds"`
	WriteTimeoutSecs *int     `yaml:"write_timeout_seconds"`
	IdleTimeoutSecs  *int     `yaml:"idle_timeout_seconds"`
	ShutdownSeconds  *int     `yaml:"shutdown_timeout_seconds"`
}

type secretKey string

const tokenSecretKey secretKey = "TOKEN_SECRET"

type secretProvider interface {
	Get(ctx context.Context, key secretKey) (string, error)
}

type envSecretProvider struct{}

func (envSecretProvider) Get(_ context.Context, key secretKey) (string, error) {
	return os.Getenv(string(key)), nil
}

type secretManagerProvider struct {
	client    *secretmanager.Client
	projectID string
	cache     sync.Map
}

func (s *secretManagerProvider) Get(ctx context.Context, key secretKey) (string, error) {
	if cached, ok := s.cache.Load(key); ok {
		return cached.(string), nil
	}

	name := os.Getenv(string(key) + "_NAME")
	if name == "" {
		name = fmt.Sprintf("projects/%s/secrets/%s/versions/latest", s.projectID, normalizeSecretID(key))
	}
	req := &secretmanagerpb.AccessSecretVersionRequest{Name: name}
	resp, err := s.client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("accessing secret %s: %w", name, err)
	}
	payload := string(resp.Payload.Data)
	s.cache.Store(key, payload)
	return payload, nil
}

func (s *secretManagerProvider) Close() error {
	return s.client.Close()
}

func normalizeSecretID(key secretKey) string {
	return strings.ToLower(strings.ReplaceAll(string(key), "_", "-"))
}

// builds config by combining ./config.yaml defaults, environment overrides,
// kube-supplied variables, and the appropriate secret backend.
func Load(ctx context.Context) (Config, error) {
	// read config.yaml file as default
	configPath := envOrDefault("CONFIG_FILE", defaultConfigFile)
	fileCfg, err := loadFileConfig(configPath)
	if err != nil {
		return Config{}, err
	}

	// get APP_ENV value with fallback
	env := resolveConfigValString(os.Getenv("APP_ENV"), fileCfg.Environment, defaultEnv)
	if strings.EqualFold(env, defaultEnv) {
		_ = godotenv.Load(".env")
		env = resolveConfigValString(os.Getenv("APP_ENV"), fileCfg.Environment, defaultEnv)
	}

	// get secret provider, for non-prod use env file
	provider, closer, err := selectSecretProvider(ctx, env)
	if err != nil {
		return Config{}, err
	}
	if closer != nil {
		defer closer()
	}

	tokenSecret, err := provider.Get(ctx, tokenSecretKey)
	if err != nil {
		return Config{}, err
	}
	if tokenSecret == "" && !strings.EqualFold(env, "production") && !strings.EqualFold(env, "staging") {
		tokenSecret = os.Getenv(string(tokenSecretKey))
	}
	if tokenSecret == "" {
		return Config{}, errors.New("token secret cannot be empty")
	}

	cfg := Config{
		Port:            resolveConfigValString(os.Getenv("PORT"), fileCfg.Port, "8000"),
		Environment:     env,
		LogLevel:        resolveConfigValString(os.Getenv("LOG_LEVEL"), fileCfg.LogLevel, "info"),
		TokenSecret:     tokenSecret,
		RateLimitRPS:    resolveConfigValFloat(os.Getenv("RATE_LIMIT_RPS"), fileCfg.RateLimitRPS, 5),
		RateLimitBurst:  resolveConfigValInt(os.Getenv("RATE_LIMIT_BURST"), fileCfg.RateLimitBurst, 10),
		ReadTimeout:     resolveConfigDuration(os.Getenv("READ_TIMEOUT_SECONDS"), fileCfg.ReadTimeoutSecs, 5),
		WriteTimeout:    resolveConfigDuration(os.Getenv("WRITE_TIMEOUT_SECONDS"), fileCfg.WriteTimeoutSecs, 10),
		IdleTimeout:     resolveConfigDuration(os.Getenv("IDLE_TIMEOUT_SECONDS"), fileCfg.IdleTimeoutSecs, 120),
		ShutdownTimeout: resolveConfigDuration(os.Getenv("SHUTDOWN_TIMEOUT_SECONDS"), fileCfg.ShutdownSeconds, 5),
		UseSecretMgr:    closer != nil,
	}

	return cfg, nil
}

func selectSecretProvider(ctx context.Context, env string) (secretProvider, func(), error) {
	switch strings.ToLower(env) {
	case "production", "prod", "staging":
		projectID := os.Getenv("GCP_PROJECT")
		if projectID == "" {
			return nil, nil, errors.New("GCP_PROJECT must be set when using secret manager")
		}
		client, err := secretmanager.NewClient(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("initializing secret manager client: %w", err)
		}
		provider := &secretManagerProvider{
			client:    client,
			projectID: projectID,
		}
		return provider, func() {
			_ = client.Close()
		}, nil
	default:
		return envSecretProvider{}, nil, nil
	}
}

func loadFileConfig(path string) (fileConfig, error) {
	var cfg fileConfig
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config file %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	return cfg, nil
}

func resolveConfigValString(envVal, fileVal, fallback string) string {
	if envVal != "" {
		return envVal
	}
	if fileVal != "" {
		return fileVal
	}
	return fallback
}

func resolveConfigValFloat(envVal string, fileVal *float64, fallback float64) float64 {
	if envVal != "" {
		if parsed, err := strconv.ParseFloat(envVal, 64); err == nil {
			return parsed
		}
	}
	if fileVal != nil {
		return *fileVal
	}
	return fallback
}

func resolveConfigValInt(envVal string, fileVal *int, fallback int) int {
	if envVal != "" {
		if parsed, err := strconv.Atoi(envVal); err == nil {
			return parsed
		}
	}
	if fileVal != nil {
		return *fileVal
	}
	return fallback
}

func resolveConfigDuration(envVal string, fileVal *int, fallback int) time.Duration {
	return time.Duration(resolveConfigValInt(envVal, fileVal, fallback)) * time.Second
}

func envOrDefault(envKey, def string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}
	return def
}
