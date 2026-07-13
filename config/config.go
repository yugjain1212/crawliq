package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// config files holds all application configuration, loaded from config.yaml
// and overridable via env variables.

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Crawler  CrawlerConfig  `mapstructure:"crawler"`
	Log      LogConfig      `mastructure:"log"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` //for "debug" and "release" Gin mode
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MinConns        int32         `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
}

// DSN builds a Postgres connection string from database config
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.Username, d.Password, d.Name, d.SSLMode,
	)

}

type CrawlerConfig struct {
	Workers          int           `mapstructure:"workers"`
	RequestTimeout   time.Duration `mapstructure:"timeout"`
	MaxRedirects     int           `mapstructure:"max_redirects"`
	UserAgent        string        `mapstructure:"user_agent"`
	PerDomainDelay   time.Duration `mapstructure:"per_domain_delay"`
	RespectRobotsTxt bool          `mapstructure:"respect_robots_txt"`
	MaxPagesPerCrawl int           `mapstructure:"max_pages_per_domain"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`  //debug info, warn, error
	Pretty bool   `mapstructure:"pretty"` // Human-readable console output vs JSON
}

/*
	Load reads configuration from config.yaml (search in given paths),

then overlays any matching env variables (including those in a .env file, if present)
env var take precedence over the YAML file, which is the standard 12 factor precedence order.
*/
func Load(configPaths ...string) (*Config, error) {
	/*Load .env into the process env if exists. It's fine if there is no .env file
	(e.g. in production where real env vars are injected directly) so we deliberately ignore that specific error */

	if err := godotenv.Load(); err != nil {
		if !isFileNotExist(err) {
			return nil, fmt.Errorf("error Loading .env file: %w", err)
		}
	}

	v := viper.New()

	setDefaults(v)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	if len(configPaths) == 0 {
		configPaths = []string{"./config", "."}
	}
	for _, p := range configPaths {
		v.AddConfigPath(p)
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		//No config.yaml found fall back to defaults + env vars only
	}

	//Allows env vars live CRAWLIQ_DATABASE_HOST to override database.host, etc.

	v.SetEnvPrefix("CRAWLIQ")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)

	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.username", "postgres")
	v.SetDefault("databse.ssl_mode", "disable")
	v.SetDefault("database.max_conns", 10)
	v.SetDefault("database.min_conns", 2)
	v.SetDefault("database.max_conn_lifetime", "30m")

	v.SetDefault("crawler.workers_count", 50)
	v.SetDefault("crawler.request_timeout", "10s")
	v.SetDefault("crawler.max_redirects", 5)
	v.SetDefault("crawler.user_agent", "CrawlIQ/1.0 (+https://github.com/yugjain1212/crawliq)")
	v.SetDefault("crawler.per_domain_delay", "0s")
	v.SetDefault("crawler.respect_robots_txt", true)
	v.SetDefault("crawler.max_pages_per_crawl", 5000)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.pretty", true)
}

// validate catches config mistakes early instead of letting surface as confusing errors deep in the crawler or DB layer later.
func (c *Config) validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database.host must not be empty")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database.name must not be empty")
	}
	if c.Database.Username == "" {
		return fmt.Errorf("database.username must not be empty")
	}
	if c.Crawler.Workers <= 0 {
		return fmt.Errorf("crawler.workers must be greater than 0")
	}
	if c.Crawler.RequestTimeout <= 0 {
		return fmt.Errorf("crawler.request_timeout must be greater than 0")
	}
	return nil

}
func isFileNotExist(err error) bool {
	return strings.Contains(err.Error(), "no such file or directory")

}
