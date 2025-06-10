package utils

import (
	"os"

	"gopkg.in/yaml.v2"
)

/*
 * Database connection configuration
 * @param Type Database type (mysql/sqlite)
 * @param DatabaseName Database name
 * @param Host Database host
 * @param Port Database port
 * @param Password Database password
 * @param User Database username
 */
type DbConfig struct {
	Type         string `yaml:"type"` // Database type, mysql or sqlite
	DatabaseName string `yaml:"databaseName"`
	Host         string `yaml:"host"`
	Port         string `yaml:"port"`
	Password     string `yaml:"password"`
	User         string `yaml:"user"`
}

/**
 * Web server configuration
 */
type ServerConfig struct {
	ListenAddr string `yaml:"listenAddr"`
	Debug      bool   `yaml:"debug"`
	Logger     bool   `yaml:"logger"`
}

/*
 * Timeout configuration
 * @param PhaseQueueDefault Default queue phase timeout (seconds)
 * @param PhaseInitDefault Default initialization phase timeout (seconds)
 * @param PhaseRunningDefault Default running phase timeout (seconds)
 * @param PhaseWholeDefault Default whole task timeout (seconds)
 */
type TimeoutConfig struct {
	PhaseQueueDefault   int `yaml:"phaseQueueDefault"`
	PhaseInitDefault    int `yaml:"phaseInitDefault"`
	PhaseRunningDefault int `yaml:"phaseRunningDefault"`
	PhaseWholeDefault   int `yaml:"phaseWholeDefault"`
}

/*
 * Authentication configuration
 * @param Enable Whether to enable authentication
 * @param Url Authentication service URL
 * @param FakeUser Mock user for testing
 */
type AuthConfig struct {
	Enable   bool   `yaml:"enable"`
	Url      string `yaml:"url"`
	FakeUser string `yaml:"fakeUser"`
}

/*
 * WeChat notification configuration
 * @param Enable Whether to enable WeChat notifications
 * @param Proxy WeChat proxy address
 * @param RobotURL WeChat robot URL
 */
type WeChatConfig struct {
	Enable   bool   `yaml:"enable"`
	Proxy    string `yaml:"proxy"`
	RobotURL string `yaml:"robot"`
}

/*
 * Redis configuration
 * @param Addr Redis address
 * @param Password Redis password
 * @param DB Redis database number
 */
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type LoggerConfig struct {
	Level  string `yaml:"level"`
	Output string `yaml:"output"`
	Format string `yaml:"format"`
}

/*
 * Main system configuration
 * @param Env Environment identifier
 * @param Db Database configuration
 * @param Redis Redis configuration
 * @param Timeout Timeout configuration
 * @param WeChat WeChat notification configuration
 * @param LokiURL Loki log service URL
 * @param Priority Task priority configuration
 */
type Config struct {
	Env     string        `yaml:"env"`
	Db      DbConfig      `yaml:"db"`
	Redis   RedisConfig   `yaml:"redis"`
	Server  ServerConfig  `yaml:"server"`
	Timeout TimeoutConfig `yaml:"timeout"`
	WeChat  WeChatConfig  `yaml:"wechat"`
	LokiURL string        `yaml:"loki"`
	Logger  LoggerConfig  `yaml:"logger"`
}

/*
 * Initialize configuration
 * @param filePath Configuration file path
 * @return error Error object
 */
func (config *Config) Init(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	return nil
}
