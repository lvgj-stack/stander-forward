package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cloudwego/hertz/pkg/common/hlog"

	"github.com/Mr-LvGJ/stander/pkg/common"
)

func DoRequest(controllerAddr, group, action, key string, req any) (any, error) {
	hlog.Debugf("do request, controllerAddr: %s, group: %s, action: %s, key: %s, req: %v", controllerAddr, group, action, key, req)
	body, _ := json.Marshal(req)
	request, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("http://%s/api/v1/%s?Action=%s", controllerAddr, group, action),
		bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set(common.KeyHeader, key)
	request.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status not equal 200 failed, controllerAddr: %s, group: %s, action: %s, response: %s",
			controllerAddr, group, action, string(bs))
	}
	hlog.Info(string(bs))
	return string(bs), nil
}
