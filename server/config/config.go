package config

import (
	"os"

	"github.com/spf13/viper"
)

var (
	Config = &Conf{}
)

type Conf struct {
	GrpcConfig        *GrpcConfig         `mapstructure:"grpc"`
	LogConfig         *LogConfig          `mapstructure:"log"`
	HttpConfig        *HttpConfig         `mapstructure:"http"`
	SystemConfig      *SystemConfig       `mapstructure:"system"`
	KafkaConfig       *KafkaConfig        `mapstructure:"kafka"`
	PaymentGrpcConfig *PaymentGrpcConfig  `mapstructure:"payment_grpc"`
	IdentityConfig    *GrpcClientConfig   `mapstructure:"identity"`
}

type GrpcClientConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type HttpConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type LogConfig struct {
	Level    string `mapstructure:"level"`
	FilePath string `mapstructure:"file_path"`
}

type GrpcConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	ConnectTimeout int    `mapstructure:"connect_timeout"`
	MaxPoolSize    int    `mapstructure:"max_pool_size"`
}

type SystemConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type KafkaConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	Brokers  []string `mapstructure:"brokers"`
	GroupID  string   `mapstructure:"group_id"`
	ClientID string   `mapstructure:"client_id"`
	Topics   []string `mapstructure:"topics"`
}

type PaymentGrpcConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

func Init() {
	workDir, _ := os.Getwd()
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(workDir + "/resources")
	viper.AddConfigPath(workDir)

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	err = viper.Unmarshal(&Config)
	if err != nil {
		panic(err)
	}
}
