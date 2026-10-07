package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type (
	Config struct {
		LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
		Server   Server `envPrefix:"SERVER_"`
		MCP      MCP    `envPrefix:"MCP_"`

		OAuth OAuth `envPrefix:"OAUTH_"`
	}

	Server struct {
		Host string `env:"HOST" envDefault:"localhost"`
		Port int    `env:"PORT" envDefault:"8080"`
	}

	MCP struct {
		Port      int    `env:"PORT" envDefault:"8060"`
		AuthToken string `env:"AUTH_TOKEN,required,notEmpty"`

		BasePath string `env:"BASE_PATH" envDefault:"/mcp/health"`
	}

	OAuth struct {
		ClientID     string   `env:"CLIENT_ID,required,notEmpty"`
		ClientSecret string   `env:"CLIENT_SECRET,required,notEmpty"`
		RedirectURL  string   `env:"REDIRECT_URL" envDefault:"http://localhost:8080/oauth/callback"`
		Scopes       []string `env:"SCOPES" envDefault:"https://www.googleapis.com/auth/googlehealth"`
		TokenFile    string   `env:"TOKEN_FILE" envDefault:"token.json"`
	}
)

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := new(Config)

	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
