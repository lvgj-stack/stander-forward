package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/common/hlog"

	"github.com/Mr-LvGJ/stander/pkg/client/typ"
	"github.com/Mr-LvGJ/stander/pkg/config"
	"github.com/Mr-LvGJ/stander/pkg/utils"
)

const (
	GostURL_Service  = "/config/services"
	GostURL_Chain    = "/config/chains"
	GostURL_Observer = "/config/observers"
)

var GostCli *gostCli

type gostCli struct {
	host string
}

func InitGostCli(ctx context.Context) {
	GostCli = &gostCli{host: "http://127.0.0.1:19123"}
	if err := GostCli.AddObservers(context.TODO(), &typ.RequestForObserverRequest{
		Name: "obs-0",
		Plugin: typ.PluginForObserverRequest{
			Type: "http",
			Addr: fmt.Sprintf("http://127.0.0.1:%d/api/v1/data?Action=ObserverNetworkTraffic", config.GetAgentConfig().Port),
		},
	}); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return
		}
		panic(err)
	}
	//c := cron.New(time.Hour)
	//go c.Do(ctx, updateCa)
}

func updateCa(_ context.Context) error {
	if err := utils.DownloadFile("https://file.byte.gs/certFile.pem", "/etc/gost", "certFile.pem"); err != nil {
		hlog.Errorf("[updateCa] download certFile.pem failed, err: %v", err)
		return err
	}
	if err := utils.DownloadFile("https://file.byte.gs/key.pem", "/etc/gost", "key.pem"); err != nil {
		hlog.Errorf("[updateCa] download key.pem failed, err: %v", err)
		return err
	}
	return nil
}

/*
*
observers:
  - name: observer-0
    plugin:
    type: grpc
    addr: 127.0.0.1:8000
    tls:
    secure: false
    serverName: example.com
*/
func (i *gostCli) AddObservers(_ context.Context, req *typ.RequestForObserverRequest) error {
	reqs, err := http.NewRequest(http.MethodPost, i.host+GostURL_Observer, utils.MustStructToReader(req))
	do, err := http.DefaultClient.Do(reqs)
	if err != nil {
		return err
	}
	bs, err := io.ReadAll(do.Body)
	if err != nil {
		return err
	}
	hlog.Infof("[AddObservers] resp: %s", string(bs))
	var resp *typ.Response
	if err := json.Unmarshal(bs, &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return errors.New(resp.Msg)
	}
	return nil
}

func (i *gostCli) AddService(ctx context.Context, req *typ.RequestForServiceRequest) error {
	req.Observer = "obs-0"
	req.Metadata = map[string]string{
		"enableStats":           "true",
		"observer.period":       "60s",
		"observer.resetTraffic": "true",
	}
	reqs, err := http.NewRequest(http.MethodPost, i.host+GostURL_Service, utils.MustStructToReader(req))
	do, err := http.DefaultClient.Do(reqs)
	if err != nil {
		return err
	}
	bs, err := io.ReadAll(do.Body)
	if err != nil {
		return err
	}
	hlog.Infof("[AddService] resp: %s", string(bs))
	var resp *typ.Response
	if err := json.Unmarshal(bs, &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return errors.New(resp.Msg)
	}
	return nil
}

func (i *gostCli) DeleteService(ctx context.Context, serviceName string) error {
	request, err := http.NewRequest(http.MethodDelete, i.host+GostURL_Service+"/"+serviceName, nil)
	do, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	bs, err := io.ReadAll(do.Body)
	if err != nil {
		return err
	}
	hlog.Infof("[DeleteService] resp: %s", string(bs))
	var resp *typ.Response
	if err := json.Unmarshal(bs, &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return errors.New(resp.Msg)
	}
	return nil
}

func (i *gostCli) UpdateService(ctx context.Context, req *typ.RequestForServiceRequest) error {
	request, err := http.NewRequest(http.MethodPut, i.host+GostURL_Service+"/"+req.Name, nil)
	do, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	bs, err := io.ReadAll(do.Body)
	if err != nil {
		return err
	}
	hlog.Infof("[UpdateService] resp: %s", string(bs))
	var resp *typ.Response
	if err := json.Unmarshal(bs, &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return errors.New(resp.Msg)
	}
	return nil
}

func (i *gostCli) AddChain(ctx context.Context, req *typ.RequestForChainRequest) error {
	request, err := http.NewRequest(http.MethodPost, i.host+GostURL_Chain, utils.MustStructToReader(req))
	do, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	bs, err := io.ReadAll(do.Body)
	if err != nil {
		return err
	}
	hlog.Infof("[AddChain] resp: %s", string(bs))
	var resp *typ.Response
	if err := json.Unmarshal(bs, &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return errors.New(resp.Msg)
	}
	return nil

}
