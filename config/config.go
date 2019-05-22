package config

import (
	"fmt"
	"github.com/go-playground/validator"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Version  string `yaml:"Version" validate:"required"`
	LogLevel string `yaml:"LogLevel" validate:"required"`

	*Telegram `yaml:"Telegram" validate:"required"`
	*DB       `yaml:"DB" validate:"required"`
}

type Telegram struct {
	Token string `yaml:"Token" validate:"required"`
}

type DB struct {
	Host     string `yaml:"Host" validate:"required"`
	Port     int    `yaml:"Port" validate:"required"`
	User     string `yaml:"User" validate:"required"`
	Password string `yaml:"Password" validate:"required"`
	Name     string `yaml:"Name" validate:"required"`
	SSL      bool   `yaml:"SSL"`
}

// Create PostgreSQL database connection string
func (db *DB) ConnectionString() string {
	uri := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s",
		db.Host, db.Port,
		db.User, db.Name,
		db.Password,
	)

	if db.SSL {
		uri += " sslmode=require"
	} else {
		uri += " sslmode=disable"
	}

	return uri
}

// Init new config with validation
func NewConfig(p string) (*Config, error) {
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(&c); err != nil {
		return nil, err
	}

	return &c, nil
}
