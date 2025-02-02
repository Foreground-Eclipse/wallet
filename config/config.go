package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		Server   ServerConfig
		Database DatabaseConfig
		Redis    RedisConfig
	}

	// ServerConfig holds the configuration for the server settings
	ServerConfig struct {
		Address string        `env:"SERVER_ADDRESS"` // The port on which the server will listen
		Timeout time.Duration `env:"SERVER_TIMEOUT"` // Timeout that decides when to reject the request
	}

	// DatabaseConfig holds the configuration for the database connection
	DatabaseConfig struct {
		Host     string `env:"DATABASE_HOST"`
		Port     string `env:"DATABASE_PORT"`
		Name     string `env:"DATABASE_NAME"`
		User     string `env:"DATABASE_USER"`
		Password string `env:"DATABASE_PASSWORD"`
		SSLMode  string `env:"DATABASE_SSLMODE"`
	}

	RedisConfig struct {
		Addr     string `env:"REDIS_ADDR"`     // The address of the database
		Password string `env:"REDIS_PASSWORD"` // The password for connecting to the database
		DB       int    `env:"REDIS_DB"`       // The name of the database
	}
)

// LoadConfig loads config with given filename

func MustLoad(filename string) *Config {
	configPath := fmt.Sprintf("./config/%s.env", filename)
	fmt.Println(os.Getwd())
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file doesnt exists %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cant read config: %s", err)
	}

	return &cfg

}
