package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	Server   ServerConfig
	MySQL    MySQLConfig
	MongoDB  MongoDBConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	Firebase FirebaseConfig
	APNS     APNSConfig
	Email    EmailConfig
	SMS      SMSConfig
	Worker   WorkerConfig
	LogLevel string
}

type ServerConfig struct {
	Port string
	Env  string
}

type MySQLConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

type MongoDBConfig struct {
	URI      string
	Database string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers       []string
	Topic         string
	ConsumerGroup string
}

type FirebaseConfig struct {
	CredentialsFile string
	ProjectID       string
}

type APNSConfig struct {
	KeyFile  string
	KeyID    string
	TeamID   string
	BundleID string
	UseSandbox bool
}

type EmailConfig struct {
	Provider    string // sendgrid, smtp
	APIKey      string
	FromEmail   string
	FromName    string
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
}

type SMSConfig struct {
	Provider    string // twilio
	AccountSID  string
	AuthToken   string
	FromNumber  string
}

type WorkerConfig struct {
	PoolSize    int
	RetryCount  int
	RetryDelay  int // seconds
}

// Load reads configuration from environment and config files
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.env", "development")
	v.SetDefault("loglevel", "info")

	v.SetDefault("mysql.host", "localhost")
	v.SetDefault("mysql.port", "3306")
	v.SetDefault("mysql.user", "quickchat")
	v.SetDefault("mysql.password", "quickchat_notification_password")
	v.SetDefault("mysql.database", "quickchat_notification")

	v.SetDefault("mongodb.uri", "mongodb://localhost:27017")
	v.SetDefault("mongodb.database", "quickchat_notifications")

	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", "6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.topic", "notifications")
	v.SetDefault("kafka.consumergroup", "notification-service")

	v.SetDefault("worker.poolsize", 10)
	v.SetDefault("worker.retrycount", 3)
	v.SetDefault("worker.retrydelay", 60)

	v.SetDefault("email.provider", "sendgrid")
	v.SetDefault("email.fromemail", "noreply@quickchat.com")
	v.SetDefault("email.fromname", "QuickChat")

	v.SetDefault("sms.provider", "twilio")

	v.SetDefault("apns.usesandbox", true)

	// Read from environment
	v.SetEnvPrefix("NOTIFICATION")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Try to read config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/notification-service")

	if err := v.ReadInConfig(); err != nil {
		// Config file not found is okay, we use defaults/env
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: v.GetString("server.port"),
			Env:  v.GetString("server.env"),
		},
		MySQL: MySQLConfig{
			Host:     v.GetString("mysql.host"),
			Port:     v.GetString("mysql.port"),
			User:     v.GetString("mysql.user"),
			Password: v.GetString("mysql.password"),
			Database: v.GetString("mysql.database"),
		},
		MongoDB: MongoDBConfig{
			URI:      v.GetString("mongodb.uri"),
			Database: v.GetString("mongodb.database"),
		},
		Redis: RedisConfig{
			Host:     v.GetString("redis.host"),
			Port:     v.GetString("redis.port"),
			Password: v.GetString("redis.password"),
			DB:       v.GetInt("redis.db"),
		},
		Kafka: KafkaConfig{
			Brokers:       v.GetStringSlice("kafka.brokers"),
			Topic:         v.GetString("kafka.topic"),
			ConsumerGroup: v.GetString("kafka.consumergroup"),
		},
		Firebase: FirebaseConfig{
			CredentialsFile: v.GetString("firebase.credentialsfile"),
			ProjectID:       v.GetString("firebase.projectid"),
		},
		APNS: APNSConfig{
			KeyFile:    v.GetString("apns.keyfile"),
			KeyID:      v.GetString("apns.keyid"),
			TeamID:     v.GetString("apns.teamid"),
			BundleID:   v.GetString("apns.bundleid"),
			UseSandbox: v.GetBool("apns.usesandbox"),
		},
		Email: EmailConfig{
			Provider:  v.GetString("email.provider"),
			APIKey:    v.GetString("email.apikey"),
			FromEmail: v.GetString("email.fromemail"),
			FromName:  v.GetString("email.fromname"),
			SMTPHost:  v.GetString("email.smtphost"),
			SMTPPort:  v.GetInt("email.smtpport"),
			SMTPUser:  v.GetString("email.smtpuser"),
			SMTPPass:  v.GetString("email.smtppass"),
		},
		SMS: SMSConfig{
			Provider:   v.GetString("sms.provider"),
			AccountSID: v.GetString("sms.accountsid"),
			AuthToken:  v.GetString("sms.authtoken"),
			FromNumber: v.GetString("sms.fromnumber"),
		},
		Worker: WorkerConfig{
			PoolSize:   v.GetInt("worker.poolsize"),
			RetryCount: v.GetInt("worker.retrycount"),
			RetryDelay: v.GetInt("worker.retrydelay"),
		},
		LogLevel: v.GetString("loglevel"),
	}

	return cfg, nil
}
