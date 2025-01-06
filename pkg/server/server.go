package server

import (
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/hertz-contrib/logger/accesslog"

	"github.com/Mr-LvGJ/stander/pkg/api"
	"github.com/Mr-LvGJ/stander/pkg/config"
)

func InitController(c *config.Config) {
	h := server.Default(
		server.WithHostPorts(":" + c.Server.Port),
	)
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return
	}
	h.Use(accesslog.New(
		accesslog.WithTimeZoneLocation(location),
	))

	hlog.SetLevel(hlog.Level(c.Server.LogLevel))
	api.InitControllerRoute(h)
	h.Spin()
}

func InitAgent(c *config.Config) {
	h := server.Default(
		server.WithHostPorts(":" + c.Server.Port),
	)
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return
	}
	h.Use(accesslog.New(
		accesslog.WithTimeZoneLocation(location),
	))
	hlog.SetLevel(hlog.Level(c.Server.LogLevel))
	api.InitAgentRoute(h)
	h.Spin()
}
