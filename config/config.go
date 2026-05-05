package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
	JWT      JWTConfig
	Logger   LoggerConfig
	Garage   GarageConfig
	Worker   WorkerConfig
	Admin    AdminConfig
}

type ServerConfig struct {
	Port         string
	Environment  string
	ReadTimeout  int
	WriteTimeout int
}

type DatabaseConfig struct {
	Path string
}

type CacheConfig struct {
	Enabled  bool
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	PrivateKeyPEM string
	PublicKeyPEM  string
	Issuer        string
	AccessTTL     int // minutes
	RefreshTTL    int // days
	JWKS        string // Keycloak-style JWKS JSON
}

type LoggerConfig struct {
	BasePath string
	Level    string
	Console  bool
}

type GarageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
}

type WorkerConfig struct {
	Enabled       bool
	CheckInterval int
	MaxRetries    int
	UploadTimeout int
	S3Prefix      string
}

type AdminConfig struct {
	Email    string
	Password string
}

func Load() *Config {
	// Set defaults
	viper.SetDefault("server.port", "3000")
	viper.SetDefault("server.environment", "development")
	viper.SetDefault("server.read_timeout", 10)
	viper.SetDefault("server.write_timeout", 10)

	viper.SetDefault("database.path", "mangosteen.db")

	viper.SetDefault("cache.enabled", false)
	viper.SetDefault("cache.addr", "localhost:6379")
	viper.SetDefault("cache.db", 0)

	viper.SetDefault("jwt.issuer", "mangosteen")
	viper.SetDefault("jwt.access_ttl", 15)
	viper.SetDefault("jwt.refresh_ttl", 7)

	viper.SetDefault("logger.base_path", "./logs")
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.console", true)

	viper.SetDefault("garage.endpoint", "localhost:3900")
	viper.SetDefault("garage.access_key", "minioadmin")
	viper.SetDefault("garage.secret_key", "minioadmin")
	viper.SetDefault("garage.bucket", "logs")
	viper.SetDefault("garage.region", "us-east-1")
	viper.SetDefault("garage.use_ssl", false)

	viper.SetDefault("worker.enabled", true)
	viper.SetDefault("worker.check_interval", 5)
	viper.SetDefault("worker.max_retries", 3)
	viper.SetDefault("worker.upload_timeout", 30)
	viper.SetDefault("worker.s3_prefix", "mangosteen/")

	viper.SetDefault("admin.email", "")
	viper.SetDefault("admin.password", "")

	// Load from .env file if exists
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	if err := viper.ReadInConfig(); err != nil {
		log.Println("No .env file found, using defaults and environment variables")
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return &Config{
		Server: ServerConfig{
			Port:         viper.GetString("server.port"),
			Environment:  viper.GetString("server.environment"),
			ReadTimeout:  viper.GetInt("server.read_timeout"),
			WriteTimeout: viper.GetInt("server.write_timeout"),
		},
		Database: DatabaseConfig{
			Path: viper.GetString("database.path"),
		},
		Cache: CacheConfig{
			Enabled:  viper.GetBool("cache.enabled"),
			Addr:     viper.GetString("cache.addr"),
			Password: viper.GetString("cache.password"),
			DB:       viper.GetInt("cache.db"),
		},
		JWT: JWTConfig{
			PrivateKeyPEM: viper.GetString("jwt.private_key"),
			PublicKeyPEM:  viper.GetString("jwt.public_key"),
			Issuer:        viper.GetString("jwt.issuer"),
			AccessTTL:     viper.GetInt("jwt.access_ttl"),
			RefreshTTL:    viper.GetInt("jwt.refresh_ttl"),
		},
		Logger: LoggerConfig{
			BasePath: viper.GetString("logger.base_path"),
			Level:    viper.GetString("logger.level"),
			Console:  viper.GetBool("logger.console"),
		},
		Garage: GarageConfig{
			Endpoint:  viper.GetString("garage.endpoint"),
			AccessKey: viper.GetString("garage.access_key"),
			SecretKey: viper.GetString("garage.secret_key"),
			Bucket:    viper.GetString("garage.bucket"),
			Region:    viper.GetString("garage.region"),
			UseSSL:    viper.GetBool("garage.use_ssl"),
		},
		Worker: WorkerConfig{
			Enabled:       viper.GetBool("worker.enabled"),
			CheckInterval: viper.GetInt("worker.check_interval"),
			MaxRetries:    viper.GetInt("worker.max_retries"),
			UploadTimeout: viper.GetInt("worker.upload_timeout"),
			S3Prefix:      viper.GetString("worker.s3_prefix"),
		},
		Admin: AdminConfig{
			Email:    viper.GetString("admin.email"),
			Password: viper.GetString("admin.password"),
		},
	}
}
