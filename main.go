package main

import (
	"log"
	"net/http"
	_ "net/http/pprof" // 匿名导入
	"strconv"

	"github.com/spf13/pflag"

	"github.com/Mr-LvGJ/stander/pkg"
	"github.com/Mr-LvGJ/stander/pkg/common"
	"github.com/Mr-LvGJ/stander/pkg/config"
)

var (
	configPath     string
	controllerAddr string
	nodeKey        string
	ip             string
	ipv6           string
	port           int32
	logLevel       int
	preferIpv6     bool
	enableUdp      bool
	listenIp       string
)

func init() {
	pflag.StringVarP(&configPath, "config-path", "c", "stander.yaml", "define config path")
	pflag.StringVarP(&controllerAddr, "controller-addr", "a", "", "controller ip + port")
	pflag.StringVarP(&nodeKey, "node-key", "k", "", "node specify key")
	pflag.StringVarP(&ip, "ip", "", "", "ip")
	pflag.StringVarP(&ipv6, "ipv6", "", "", "ipv6")
	pflag.StringVarP(&listenIp, "listen-ip", "", "", "listen ip")
	pflag.Int32VarP(&port, "port", "p", 18123, "http port")
	pflag.IntVarP(&logLevel, "log-level", "", 3, "log level")
	pflag.BoolVarP(&preferIpv6, "prefer-ipv6", "", false, "prefer ipv6")
	pflag.BoolVarP(&enableUdp, "enable-udp", "", true, "prefer ipv6")
	pflag.Parse()
}

func main() {
	c := &config.Config{
		Server: &config.Server{},
		Agent:  &config.Agent{},
	}
	go func() {
		if err := http.ListenAndServe(":48123", nil); err != nil {
			log.Println(err)
		}
	}()
	if controllerAddr != "" {
		if nodeKey == "" {
			panic("empty node key")
		}
		c.Server.NodeRole = string(common.Agent)
		c.Server.Port = strconv.Itoa(int(port))
		c.Server.LogLevel = logLevel
		c.Agent.ControllerAddr = controllerAddr
		c.Agent.NodeKey = nodeKey
		c.Agent.IP = ip
		c.Agent.IPv6 = ipv6
		c.Agent.Port = port
		c.Agent.PreferIpv6 = preferIpv6
		c.Agent.EnableUdp = enableUdp
		c.Agent.ListenIp = listenIp
		config.SetConfig(c)
		pkg.RunAgent(c)
	} else {
		c = config.InitConfig(configPath)
		pkg.RunController(c)
	}
}
