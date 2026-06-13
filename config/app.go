package config

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/spf13/viper"
)

type S3Config struct {
	AccessKey string
	SecretKey string
}

type SafeConvert struct {
	AccessToken string
	URL         string
}

type Config struct {
	DatabaseURL string
	KongURL     string
	JWTKeys     jwk.Set
	S3          S3Config
	SafeConvert SafeConvert
}

var (
	instance *Config
	once     sync.Once
	loadErr  error
)

func MustGet() *Config {
	if cfg, err := Get(); err != nil {
		panic(fmt.Sprintf("get config: %s", err.Error()))
	} else {
		return cfg
	}
}

func Get() (*Config, error) {
	if instance == nil {
		return nil, errors.New("config is nil (was `config.Init` called?)")
	}
	return instance, nil
}

const EnvPrefix = "PROJDOCS"

func Init() (*Config, error) {
	once.Do(func() {
		v := viper.New()

		v.SetEnvPrefix(EnvPrefix)

		v.SetDefault("DATABASE_URL", "")
		v.SetDefault("JWT_KEYS", "")

		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()

		// jwk.ParseString expects a JWKS object {"keys":[...]}, not a bare array
		// Wrap if the value is a bare JSON array
		rawKeySet := strings.TrimSpace(v.GetString("JWT_KEYS"))
		if strings.HasPrefix(rawKeySet, "[") {
			rawKeySet = `{"keys":` + rawKeySet + `}`
		}
		keySet, err := jwk.ParseString(rawKeySet)
		if err != nil {
			loadErr = fmt.Errorf("config: parsing %s_JWT_KEYS: %w", EnvPrefix, err)
			return
		}

		cfg := &Config{
			KongURL:     v.GetString("KONG_URL"),
			DatabaseURL: v.GetString("DATABASE_URL"),
			JWTKeys:     keySet,
			S3: S3Config{
				AccessKey: v.GetString("SUPABASE_S3_ACCESS_KEY"),
				SecretKey: v.GetString("SUPABASE_S3_SECRET_KEY"),
			},
			SafeConvert: SafeConvert{
				AccessToken: v.GetString("SAFE_CONVERT_ACCESS_TOKEN"),
				URL:         v.GetString("SAFE_CONVERT_URL"),
			},
		}

		if err := cfg.validate(); err != nil {
			loadErr = err
			return
		}

		instance = cfg
	})

	return instance, loadErr
}

func (c *Config) validate() error {

	if c.SafeConvert.AccessToken == "" {
		return fmt.Errorf("validation: %s_SAFE_CONVERT_ACCESS_TOKEN is required", EnvPrefix)
	}

	if c.SafeConvert.URL == "" {
		return fmt.Errorf("validation: %s_SAFE_CONVERT_URL is required", EnvPrefix)
	}

	if c.DatabaseURL == "" {
		return fmt.Errorf("validation: %s_DATABASE_URL is required", EnvPrefix)
	}

	if c.KongURL == "" {
		return fmt.Errorf("validation: %s_KONG_URL is required", EnvPrefix)
	}

	if c.JWTKeys == nil {
		return fmt.Errorf("validation: %s_JWT_KEYS is unexpectedly nil", EnvPrefix)
	} else if c.JWTKeys.Len() < 1 {
		return fmt.Errorf("validation: %s_JWT_KEYS contains 0 keys (expected at least 1 key)", EnvPrefix)
	}

	if c.S3.AccessKey == "" {
		return fmt.Errorf("validation: %s_SUPABASE_S3_ACCESS_KEY is unexpectedly empty", EnvPrefix)
	}
	if c.S3.SecretKey == "" {
		return fmt.Errorf("validation: %s_SUPABASE_S3_SECRET_KEY is unexpectedly empty", EnvPrefix)
	}

	return nil
}
