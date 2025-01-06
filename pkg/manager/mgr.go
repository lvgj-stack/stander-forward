package manager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gorm.io/gorm"

	"github.com/Mr-LvGJ/stander/pkg/client"
	"github.com/Mr-LvGJ/stander/pkg/common"
	"github.com/Mr-LvGJ/stander/pkg/config"
	"github.com/Mr-LvGJ/stander/pkg/connector"
	"github.com/Mr-LvGJ/stander/pkg/connector/udp_chain_connector"
	"github.com/Mr-LvGJ/stander/pkg/model/dal"
	"github.com/Mr-LvGJ/stander/pkg/model/entity"
	"github.com/Mr-LvGJ/stander/pkg/service/req"
	"github.com/Mr-LvGJ/stander/pkg/service/resp"
	"github.com/Mr-LvGJ/stander/pkg/utils"
)

type Manager struct {
	srcConnMap *sync.Map
}

var mgr = &Manager{
	srcConnMap: &sync.Map{},
}

func Init() (*Manager, error) {
	ctx := context.TODO()
	chains, err := dal.Q.Chain.WithContext(ctx).Find()
	if err != nil {
		return nil, err
	}
	wg1 := sync.WaitGroup{}
	for _, chain := range chains {
		hlog.Infof("init chain: %v", chain)
		wg1.Add(1)
		node, err := dal.Q.Node.WithContext(ctx).Where(dal.Node.ID.Eq(chain.NodeID)).First()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		tNode := node
		tChain := chain
		go func(node *entity.Node, chain *entity.Chain) {
			for {
				r := &req.AddChainReq{
					Port:      *chain.Port,
					ChainType: *chain.Protocol,
				}
				_, err = client.DoRequest(fmt.Sprintf("%s:%d", *node.IP, *node.Port), "chain", "AddChain", *node.Key, r)
				if err != nil {
					if strings.Contains(err.Error(), "address already in use") {
						hlog.Infof("CHAIN agent port already in use, begin to delete port: %s", *chain.Port)
						dr := &req.DelChainReq{
							Port: *chain.Port,
						}
						_, _ = client.DoRequest(fmt.Sprintf("%s:%d", *node.IP, *node.Port), "chain", "DeleteChain", *node.Key, dr)
					}
					hlog.Error(err)
					time.Sleep(3 * time.Second)
					continue
				}
				break
			}
			wg1.Done()
		}(tNode, tChain)
	}
	wg1.Wait()
	rules, err := dal.Q.Rule.WithContext(ctx).Find()
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		hlog.Infof("init rule: %v", rule)
		wg1.Add(1)
		node, err := dal.Q.Node.WithContext(ctx).Where(dal.Node.ID.Eq(*rule.NodeID)).First()
		if err != nil {
			return nil, err
		}
		chain, err := dal.Q.Chain.WithContext(ctx).Where(dal.Chain.ID.Eq(*rule.ChainID)).First()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		tNode := node
		tChain := chain
		tRule := rule
		go func() {
			for {
				chainAddr := ""
				if tChain != nil {
					chainAddr = utils.GenIpAndPort(*tChain.IP, *tChain.Port)
				}
				r := &req.AddRuleReq{
					ListenPort: *tRule.ListenPort,
					ChainAddr:  chainAddr,
					RemoteAddr: *tRule.RemoteAddr,
					ChainType:  *tRule.Protocol,
				}
				hlog.Infof("RULE add port: %d", *tRule.ListenPort)
				_, err = client.DoRequest(fmt.Sprintf("%s:%d", *tNode.IP, *tNode.Port), "rule", "AddRule", *tNode.Key, r)
				if err != nil {
					if strings.Contains(err.Error(), "address already in use") {
						hlog.Infof("RULE agent port already in use, begin to delete port: %d", *tRule.ListenPort)
						dr := &req.DelRuleReq{
							Port: *tRule.ListenPort,
						}
						_, _ = client.DoRequest(fmt.Sprintf("%s:%d", *node.IP, *node.Port), "rule", "DeleteRule", *node.Key, dr)
					}
					hlog.Error(err)
					time.Sleep(3 * time.Second)
					continue
				}
				break
			}
			wg1.Done()
		}()
	}
	wg1.Wait()

	return mgr, nil
}

func InitAgent(cfg *resp.RegisterNodeResp) {
	hlog.Infof("init agent, config: %v", cfg)
	for _, chain := range cfg.Chains {
		if err := AddChain(chain.Port, common.ConnectorType(chain.ChainType)); err != nil {
			panic(err)
		}
	}
	for _, rule := range cfg.Rules {
		if err := AddRule(config.GetAgentConfig().ListenIp+":"+strconv.Itoa(int(rule.ListenPort)), rule.ChainAddr, rule.RemoteAddr, common.ConnectorType(rule.ChainType)); err != nil {
			panic(err)
		}
	}
}

func AddRule(src, chain, raddr string, typ common.ConnectorType) error {
	switch typ {
	case common.TCPConnector:
		ctr := &connector.TCPConnector{}
		if err := ctr.InitConfig(src, chain, raddr); err != nil {
			return err
		}
		go ctr.Listen()
		ctr.StoreConn(mgr.srcConnMap)

		if config.GetAgentConfig().EnableUdp {
			forwarder, err := connector.NewUDPForwarder(src, raddr, 30*time.Second)
			if err != nil {
				log.Fatalf("创建转发器失败: %v", err)
			}
			go func() {
				if err := forwarder.Start(); err != nil {
					log.Fatalf("启动转发器失败: %v", err)
				}
			}()
			mgr.srcConnMap.Store(forwarder.GetStoreConnKey(ctr.Src.Port), forwarder)
		}
	case common.TLSConnector:
		ctr := &connector.TLSConnector{}
		if err := ctr.InitConfig(src, chain, raddr, "byte.gs", nil, nil, nil); err != nil {
			return err
		}
		go ctr.Listen()
		ctr.StoreConn(mgr.srcConnMap)
		if config.GetAgentConfig().EnableUdp {
			//uc := &connector.UDPConnector{}
			//if err := uc.InitConfig(src, chain, raddr); err != nil {
			//	return err
			//}
			//hlog.Debugf("add rule: %v", uc)
			//go uc.Listen()
			//uc.StoreConn(mgr.srcConnMap)
			server, err := udp_chain_connector.NewProxyServer(src, chain, raddr, 30*time.Second)
			if err != nil {
				return err
			}
			go server.Start()
			ss := strings.Split(ctr.Src, ":")
			port, _ := strconv.Atoi(ss[1])
			mgr.srcConnMap.Store(server.GetStoreConnKey(port), server)
		}
	}
	return nil
}

func AddChain(port int32, typ common.ConnectorType) error {
	switch typ {
	case common.TCPConnector:
		ctr := &connector.TCPConnector{}
		if err := ctr.InitConfig(":"+strconv.Itoa(int(port)), "", ""); err != nil {
			return err
		}
		go ctr.Listen()
		ctr.StoreConn(mgr.srcConnMap)
	case common.TLSConnector:
		ctr := &connector.TLSConnector{}
		if err := ctr.InitConfig(":"+strconv.Itoa(int(port)), "", "", "byte.gs", nil, nil, nil); err != nil {
			return err
		}
		go ctr.Listen()
		ctr.StoreConn(mgr.srcConnMap)

		if config.GetAgentConfig().EnableUdp {
			//uc := &connector.UDPConnector{}
			//if err := uc.InitConfig(":"+strconv.Itoa(int(port)), "", ""); err != nil {
			//	return err
			//}
			//hlog.Debugf("add rule: %v", uc)
			//go uc.Listen()
			//uc.StoreConn(mgr.srcConnMap)

			server, err := udp_chain_connector.NewProxyServer(":"+strconv.Itoa(int(port)), "", "", 30*time.Second)
			if err != nil {
				return err
			}
			go server.Start()
			ss := strings.Split(ctr.Src, ":")
			port, _ := strconv.Atoi(ss[1])
			mgr.srcConnMap.Store(server.GetStoreConnKey(port), server)
		}
	}

	return nil
}

func DelPort(port int32) error {
	mgr.srcConnMap.Range(func(key, value any) bool {
		hlog.Info(key)
		return true
	})
	k := strconv.Itoa(int(port))
	v, ok := mgr.srcConnMap.Load(k)
	if !ok {
		hlog.Info("port conn not found", ", port: ", port)
		return nil
	}
	mgr.srcConnMap.Delete(k)
	c := v.(connector.Close)
	err := c.Close()
	if err != nil {
		hlog.Errorf("close conn failed, port: %d, err: %v", port, err)
	}
	udpK := (&connector.UDPConnector{}).GetStoreConnKey(int(port))
	v, ok = mgr.srcConnMap.Load(udpK)
	if !ok {
		hlog.Info("udp port conn not found", ", port: ", port)
		return nil
	}
	mgr.srcConnMap.Delete(udpK)
	c = v.(connector.Close)
	err = c.Close()
	if err != nil {
		hlog.Errorf("udp close conn failed, port: %d, err: %v", port, err)
	}
	return nil
}

func (m *Manager) Shutdown() {
	mgr.srcConnMap.Range(func(key, value any) bool {
		c := value.(connector.Close)
		c.Close()
		return true
	})
}
