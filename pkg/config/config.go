package config

import (
	"github.com/spf13/viper"

	"github.com/Mr-LvGJ/stander/pkg/common"
)

var c *Config

type Config struct {
	Server      *Server
	EnableRelay bool
	Relays      []Relay
	Database    *Database

	// for agent
	Agent *Agent
}

type Agent struct {
	ControllerAddr string
	NodeKey        string
	IP             string
	ManagerIp      string
	IPv6           string
	Port           int32
	PreferIpv6     bool
	EnableUdp      bool
	EnableGost     bool
	ListenIp       string
}

type Database struct {
	Username string
	Password string
	DBName   string
	Addr     string
}

type Server struct {
	Port     string
	NodeRole string
	LogLevel int
}

type Relay struct {
	Name          string
	ConnectorType common.ConnectorType
	Src           string
	Chain         string
	RAddr         string
}

func (r Relay) String() string {
	return r.Name + "#" + r.Src + "#" + r.Chain + "#" + r.RAddr
}

func InitConfig(configPath string) *Config {
	c = &Config{}
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
	if err := viper.Unmarshal(&c); err != nil {
		panic(err)
	}
	return c
}

func SetConfig(cc *Config) {
	c = cc
}

func GetRole() string {
	return c.Server.NodeRole
}

func GetKey() string {
	return c.Agent.NodeKey
}

func GetAgentConfig() *Agent {
	return c.Agent
}
