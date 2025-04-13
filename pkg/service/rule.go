package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gorm.io/gen"
	"gorm.io/gorm"

	"github.com/Mr-LvGJ/stander/pkg/client"
	"github.com/Mr-LvGJ/stander/pkg/common"
	"github.com/Mr-LvGJ/stander/pkg/config"
	"github.com/Mr-LvGJ/stander/pkg/manager"
	"github.com/Mr-LvGJ/stander/pkg/model/dal"
	"github.com/Mr-LvGJ/stander/pkg/model/entity"
	error2 "github.com/Mr-LvGJ/stander/pkg/service/error"
	req2 "github.com/Mr-LvGJ/stander/pkg/service/req"
	"github.com/Mr-LvGJ/stander/pkg/service/resp"
	"github.com/Mr-LvGJ/stander/pkg/utils"
)

func RuleSrv(c context.Context, ctx *app.RequestContext) {
	action := ctx.Query("Action")
	switch action {
	case "AddRule":
		rule, err := addRule(c, ctx)
		error2.WriteResponse(ctx, err, rule)
	case "DeleteRule":
		rule, err := delRule(c, ctx)
		error2.WriteResponse(ctx, err, rule)
	case "ListRules":
		rule, err := listRule(c, ctx)
		error2.WriteResponse(ctx, err, rule)
	case "ModifyRule":
		rule, err := modifyRule(c, ctx)
		error2.WriteResponse(ctx, err, rule)
	case "TestRule":
		rule, err := testRule(c, ctx)
		error2.WriteResponse(ctx, err, rule)
	default:
		error2.WriteResponse(ctx, errors.New("action not found"), nil)
	}
}

func testRule(ctx context.Context, c *app.RequestContext) (*resp.TestRuleResp, error) {
	req := req2.TestRuleReq{}
	if err := c.BindAndValidate(&req); err != nil {
		return nil, err
	}
	if config.GetRole() == string(common.Agent) {
		ping, err := utils.HandleTcpping(req.Destination)
		if err != nil {
			return nil, err
		}
		return &resp.TestRuleResp{Ping: ping}, nil
	}

	res := &resp.TestRuleResp{}

	rule, err := dal.Rule.WithContext(ctx).Where(dal.Rule.ID.Eq(req.ID)).Preload(dal.Rule.Chain, dal.Rule.Node).First()
	if err != nil {
		return nil, err
	}

	node := rule.Node
	// inbound ping chain
	if rule.Chain.IP != nil {
		chainInAddr := utils.GenIpAndPort(*rule.Chain.IP, *rule.Chain.Port)
		res.InboundName = *rule.Node.NodeName
		res.InboundTo = chainInAddr
		req.Destination = chainInAddr
		res1, err := client.DoRequest(fmt.Sprintf("%s:%d", node.ManagerIP, *node.Port), "rule", "TestRule", *node.Key, req)
		if err != nil {
			return nil, err
		}
		r := resp.RawResponse[resp.TestRuleResp]{}
		if err := json.Unmarshal([]byte(res1.(string)), &r); err != nil {
			return nil, err
		}
		if r.Error != nil {
			return nil, r.Error
		}
		res.InboundPing = r.Result.Ping

		chainNode, err := dal.Node.WithContext(ctx).Where(dal.Node.ID.Eq(rule.Chain.NodeID)).First()
		if err != nil {
			return nil, err
		}
		// chain ping destination
		destinationAddr := *rule.RemoteAddr
		req.Destination = destinationAddr
		res2, err := client.DoRequest(fmt.Sprintf("%s:%d", chainNode.ManagerIP, *chainNode.Port), "rule", "TestRule", *chainNode.Key, req)
		if err != nil {
			return nil, err
		}
		r2 := resp.RawResponse[resp.TestRuleResp]{}
		if err := json.Unmarshal([]byte(res2.(string)), &r2); err != nil {
			return nil, err
		}
		if r2.Error != nil {
			return nil, r2.Error
		}
		res.OutboundName = *chainNode.NodeName
		res.OutboundTo = destinationAddr
		res.OutboundPing = r2.Result.Ping
	} else {
		destinationAddr := *rule.RemoteAddr
		res.OutboundName = *rule.Node.NodeName
		res.OutboundTo = destinationAddr
		req.Destination = destinationAddr
		res1, err := client.DoRequest(fmt.Sprintf("%s:%d", node.ManagerIP, *node.Port), "rule", "TestRule", *node.Key, req)
		if err != nil {
			return nil, err
		}
		r := resp.RawResponse[resp.TestRuleResp]{}
		if err := json.Unmarshal([]byte(res1.(string)), &r); err != nil {
			return nil, err
		}
		if r.Error != nil {
			return nil, r.Error
		}
		res.OutboundPing = r.Result.Ping
	}
	return res, nil
}

func addRule(c context.Context, ctx *app.RequestContext) (*resp.AddRuleResp, error) {
	req := req2.AddRuleReq{}
	if err := ctx.BindAndValidate(&req); err != nil {
		return nil, err
	}
	if req.ChainType == "" {
		req.ChainType = string(common.TCPConnector)
	}

	if config.GetRole() == string(common.Agent) {
		if err := manager.AddRule(config.GetAgentConfig().ListenIp+":"+strconv.Itoa(int(req.ListenPort)), req.ChainAddr, req.RemoteAddr, common.ConnectorType(req.ChainType)); err != nil {
			hlog.Errorf("add rule failed, err: %s", err.Error())
			return nil, err
		}
		return &resp.AddRuleResp{}, nil
	}
	uid := ctx.GetInt32(common.HeaderUserKey)

	chain, err := dal.Q.Chain.WithContext(c).Where(dal.Chain.ID.Eq(req.ChainId)).First()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if chain != nil {
		req.ChainAddr = utils.GenIpAndPort(*chain.IP, *chain.Port)
	}
	hlog.Infof("add direct forward, listen port: %d, raddr: %s", req.ListenPort, req.RemoteAddr)
	node, err := dal.Q.Node.WithContext(c).Where(dal.Node.ID.Eq(req.NodeId)).First()
	if err != nil {
		return nil, err
	}
	_, err = client.DoRequest(fmt.Sprintf("%s:%d", node.ManagerIP, *node.Port), "rule", "AddRule", *node.Key, req)
	if err != nil {
		return nil, err
	}
	if err := dal.Q.Rule.WithContext(c).Create(&entity.Rule{
		RuleName:   &req.RuleName,
		ListenPort: &req.ListenPort,
		RemoteAddr: &req.RemoteAddr,
		NodeID:     &req.NodeId,
		ChainID:    &req.ChainId,
		Protocol:   &req.ChainType,
		UserID:     &uid,
	}); err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return nil, errors.New("the specified rule already exist")
		}
		return nil, err
	}
	return &resp.AddRuleResp{}, nil
}

func delRule(c context.Context, ctx *app.RequestContext) (*resp.DelRuleResp, error) {
	req := req2.DelRuleReq{}
	if err := ctx.BindAndValidate(&req); err != nil {
		return nil, err
	}

	if config.GetRole() == string(common.Agent) {
		hlog.Infof("del rule, port: %d", req.Port)
		if err := manager.DelPort(req.Port, manager.ServicePortType); err != nil {
			return nil, err
		}
		return &resp.DelRuleResp{}, nil
	}

	uid := ctx.GetInt32(common.HeaderUserKey)
	if ctx.GetString(common.HeaderRoleKey) != "SUPER_ADMIN" {
		_, err := dal.Rule.WithContext(c).Where(dal.Rule.UserID.Eq(uid), dal.Rule.ID.Eq(req.ID)).First()
		if err != nil {
			return nil, err
		}
	}

	rule, err := dal.Rule.WithContext(c).Where(dal.Rule.ID.Eq(req.ID)).First()
	if err != nil {
		return nil, err
	}

	node, err := dal.Q.Node.WithContext(c).Where(dal.Node.ID.Eq(*rule.NodeID)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if _, err := dal.Q.Rule.WithContext(c).Where(dal.Rule.ID.Eq(req.ID)).Delete(&entity.Rule{}); err != nil {
				return nil, err
			}
			return &resp.DelRuleResp{RuleId: req.ID}, nil
		}
		return nil, err
	}
	req.Port = *rule.ListenPort
	_, err = client.DoRequest(fmt.Sprintf("%s:%d", node.ManagerIP, *node.Port), "rule", "DeleteRule", *node.Key, req)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			hlog.Infof("del rule failed, err: %s", err.Error())
		} else {
			return nil, err
		}
	}
	if _, err := dal.Q.Rule.WithContext(c).Where(dal.Rule.ID.Eq(req.ID)).Delete(&entity.Rule{}); err != nil {
		return nil, err
	}
	return &resp.DelRuleResp{RuleId: req.ID}, nil
}

func listRule(c context.Context, ctx *app.RequestContext) (*resp.ListRuleResp, error) {
	req := req2.ListRuleReq{}
	if err := ctx.BindAndValidate(&req); err != nil {
		return nil, err
	}

	if config.GetRole() == string(common.Agent) {
		return &resp.ListRuleResp{}, nil
	}
	uid := ctx.GetInt32(common.HeaderUserKey)

	var q []gen.Condition
	if ctx.GetString(common.HeaderRoleKey) != "SUPER_ADMIN" {
		q = append(q, dal.Rule.UserID.Eq(uid))
	}
	if req.RuleName != "" {
		q = append(q, dal.Rule.RuleName.Like(fmt.Sprintf("%%%s%%", req.RuleName)))
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}
	if req.PageNo == 0 {
		req.PageNo = 1
	}
	if req.PageNo == -1 {
		req.PageSize = 1000
	}
	od := dal.Rule.UpdatedAt.Desc()
	if req.Asc {
		od = dal.Rule.UpdatedAt.Asc()
	}
	if req.OrderBy != "" {
		name, ok := dal.Rule.GetFieldByName(req.OrderBy)
		if ok {
			od = name.Desc()
			if req.Asc {
				od = name.Asc()
			}
		}
	}
	rules, cnt, err := dal.Rule.WithContext(c).Where(q...).
		Preload(dal.Rule.Node, dal.Rule.Chain).
		Order(od).
		FindByPage(int((req.PageNo-1)*req.PageSize), int(req.PageSize))
	if err != nil {
		return nil, err
	}

	return &resp.ListRuleResp{Rules: rules, TotalCount: cnt}, nil
}

func modifyRule(ctx context.Context, c *app.RequestContext) (*resp.ModifyRuleResp, error) {
	req := req2.ModifyRuleReq{}
	if err := c.BindAndValidate(&req); err != nil {
		return nil, err
	}

	var needCallAgent bool

	if config.GetRole() == string(common.Agent) {
		if err := manager.DelPort(req.OldListenPort, manager.ServicePortType); err != nil {
			hlog.Errorf("del rule failed, port: %d, err: %s", req.OldListenPort, err.Error())
		}
		if err := manager.AddRule(config.GetAgentConfig().ListenIp+":"+strconv.Itoa(int(req.ListenPort)), req.ChainAddr, req.RemoteAddr, common.ConnectorType(req.ChainType)); err != nil {
			return nil, err
		}
		return &resp.ModifyRuleResp{}, nil
	}

	rule, err := dal.Rule.WithContext(ctx).Where(dal.Rule.ID.Eq(req.ID)).Preload(dal.Rule.Node, dal.Rule.Chain).First()
	if err != nil {
		return nil, err
	}
	node := rule.Node
	originChain := rule.Chain

	updateRule := entity.Rule{}
	agentModifyReq := req2.ModifyRuleReq{
		ListenPort:    *rule.ListenPort,
		OldListenPort: *rule.ListenPort,
		RemoteAddr:    *rule.RemoteAddr,
		ChainType:     *rule.Protocol,
	}

	if originChain.IP != nil && originChain.Port != nil {
		agentModifyReq.ChainAddr = utils.GenIpAndPort(*originChain.IP, *originChain.Port)
	}

	if req.ChainId != 0 && req.ChainId != originChain.ID {
		needCallAgent = true
		chain, err := dal.Chain.WithContext(ctx).Where(dal.Chain.ID.Eq(req.ChainId)).First()
		if err != nil {
			return nil, err
		}
		agentModifyReq.ChainAddr = utils.GenIpAndPort(*chain.IP, *chain.Port)
		agentModifyReq.ChainType = *chain.Protocol
		updateRule.ChainID = &req.ChainId
	}
	if req.ListenPort != 0 && req.ListenPort != *rule.ListenPort {
		needCallAgent = true
		agentModifyReq.ListenPort = req.ListenPort
		updateRule.ListenPort = &req.ListenPort
	}

	if req.RuleName != "" {
		updateRule.RuleName = &req.RuleName
	}
	if req.RemoteAddr != "" && req.RemoteAddr != *rule.RemoteAddr {
		needCallAgent = true
		agentModifyReq.RemoteAddr = req.RemoteAddr
		updateRule.RemoteAddr = &req.RemoteAddr
	}

	if needCallAgent {
		_, err = client.DoRequest(utils.GenIpAndPort(node.ManagerIP, *node.Port), "rule", "ModifyRule", *node.Key, agentModifyReq)
		if err != nil {
			return nil, err
		}
	}
	_, err = dal.Rule.WithContext(ctx).Where(dal.Rule.ID.Eq(req.ID)).Updates(updateRule)
	if err != nil {
		return nil, err
	}

	return &resp.ModifyRuleResp{}, nil
}
