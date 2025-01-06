package pkg

import (
	"encoding/json"

	"github.com/cloudwego/hertz/pkg/common/hlog"

	"github.com/Mr-LvGJ/stander/pkg/client"
	"github.com/Mr-LvGJ/stander/pkg/config"
	"github.com/Mr-LvGJ/stander/pkg/manager"
	"github.com/Mr-LvGJ/stander/pkg/model"
	"github.com/Mr-LvGJ/stander/pkg/server"
	"github.com/Mr-LvGJ/stander/pkg/service/req"
	"github.com/Mr-LvGJ/stander/pkg/service/resp"
	"github.com/Mr-LvGJ/stander/pkg/utils"
)

func RunController(c *config.Config) {
	model.InitMysql(c.Database)
	if c.EnableRelay {
		go func() {
			mgr, err := manager.Init()
			if err != nil {
				panic(err)
			}
			defer mgr.Shutdown()
		}()
	}
	server.InitController(c)
}

func RunAgent(c *config.Config) {
	req := &req.RegisterNodeReq{}
	req.Port = 8123
	req.Ipv4 = utils.GetOutBoundIPv4()
	req.Ipv6 = utils.GetOutBoundIPv6()
	req.IP = req.Ipv4
	req.PreferIpv6 = c.Agent.PreferIpv6

	if c.Agent.Port != 0 {
		req.Port = c.Agent.Port
	}
	if c.Agent.IP != "" {
		req.IP = c.Agent.IP
		req.Ipv4 = c.Agent.IP
	}
	if c.Agent.IPv6 != "" {
		req.Ipv6 = c.Agent.IPv6
	}
	if req.PreferIpv6 {
		req.IP = req.Ipv6
	}

	res, err := client.DoRequest(c.Agent.ControllerAddr, "node", "RegisterNode", c.Agent.NodeKey, req)
	if err != nil {
		panic(err)
	}
	hlog.Infof("response: %s", res)
	initInfo := resp.RawResponse[resp.RegisterNodeResp]{}
	if err := json.Unmarshal([]byte(res.(string)), &initInfo); err != nil {
		panic(err)
	}
	manager.InitAgent(&initInfo.Result)
	server.InitAgent(c)
}
