package service

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/Mr-LvGJ/stander/pkg/service/req"
	"github.com/Mr-LvGJ/stander/pkg/service/resp"
)

func ChainSrv(c context.Context, ctx *app.RequestContext) {
	action := ctx.Query("Action")
	switch action {
	case "AddChain":
		resp, err := addChain(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	case "DeleteChain":
		resp, err := delChain(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	case "ListChains":
		resp, err := listChain(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	case "EditChain":
		resp, err := editChain(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	default:
		error2.WriteResponse(ctx, errors.New("action not found"), nil)
	}
}

func addChain(c context.Context, ctx *app.RequestContext) (*resp.AddChainResp, error) {
	r := req.AddChainReq{}
	if err := ctx.BindAndValidate(&r); err != nil {
		return nil, err
	}
	if r.ChainType == "" {
		r.ChainType = string(common.TCPConnector)
	}
	if config.GetRole() == common.Agent.String() {
		if err := manager.AddChain(r.Port, common.ConnectorType(r.ChainType)); err != nil {
			hlog.Errorf("add chain failed, err: %s", err.Error())
			return nil, err
		}
		return &resp.AddChainResp{}, nil
	}
	node, err := dal.Q.Node.WithContext(c).Where(dal.Node.ID.Eq(r.NodeId)).First()
	if err != nil {
		return nil, err
	}

	chainIp := *node.IP
	if r.PreferIpv6 {
		chainIp = *node.Ipv6
	}

	_, err = dal.Chain.WithContext(c).Where(
		dal.Chain.Port.Eq(r.Port),
		dal.Chain.NodeID.Eq(r.NodeId),
		dal.Chain.IP.Eq(chainIp)).First()
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("chain port exist")
	}
	cnt, err := dal.Q.Chain.WithContext(c).Where(dal.Chain.NodeID.Eq(node.ID), dal.Chain.Port.Eq(r.Port)).Count()
	if err != nil {
		return nil, err
	}
	if cnt == 0 {
		_, err = client.DoRequest(fmt.Sprintf("%s:%d", *node.IP, *node.Port), "chain", "AddChain", *node.Key, r)
		if err != nil {
			return nil, err
		}
	}

	if err := dal.Q.Chain.WithContext(c).Create(&entity.Chain{
		ChainName: &r.Name,
		Port:      &r.Port,
		IP:        &chainIp,
		NodeID:    r.NodeId,
		Protocol:  &r.ChainType,
	}); err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return nil, errors.New("the specified chain already exist")
		}
		return nil, err
	}
	chain, _ := dal.Chain.WithContext(c).Where(dal.Chain.NodeID.Eq(r.NodeId), dal.Chain.Port.Eq(r.Port), dal.Chain.IP.Eq(chainIp)).First()
	userId := ctx.GetInt32(common.HeaderUserKey)
	if err := dal.UserRoleChainMapping.WithContext(c).Create(&entity.UserRoleChainMapping{
		UserID:  &userId,
		ChainID: int32(chain.ID),
	}); err != nil {
		return nil, err
	}

	return &resp.AddChainResp{}, nil
}

func delChain(c context.Context, ctx *app.RequestContext) (*resp.DelChainResp, error) {
	r := req.DelChainReq{}
	if err := ctx.BindAndValidate(&r); err != nil {
		return nil, err
	}
	if config.GetRole() == common.Agent.String() {
		if err := manager.DelPort(r.Port); err != nil {
			hlog.Errorf("delete port failed, err: %s", err.Error())
			return nil, err
		}
		return &resp.DelChainResp{}, nil
	}
	if ctx.GetString(common.HeaderRoleKey) != "SUPER_ADMIN" {
		chainM, err := dal.UserRoleChainMapping.WithContext(c).
			Where(dal.UserRoleChainMapping.UserID.Eq(ctx.GetInt32(common.HeaderUserKey)), dal.UserRoleChainMapping.ChainID.Eq(int32(r.ID))).
			Preload(dal.UserRoleChainMapping.Chain).First()
		if err != nil {
			return nil, err
		}
		chain := chainM.Chain
		cnt, err := dal.Q.Chain.WithContext(c).Where(dal.Chain.NodeID.Eq(chain.NodeID), dal.Chain.Port.Eq(*chain.Port)).Count()
		if err != nil {
			return nil, err
		}
		node, _ := dal.Node.WithContext(c).Where(dal.Node.ID.Eq(chain.NodeID)).First()
		// may ipv4 ipv6 all use one port, so only count == 1, delete port
		if cnt == 1 {
			r.Port = *chain.Port
			_, err = client.DoRequest(fmt.Sprintf("%s:%d", *node.IP, *node.Port), "chain", "DeleteChain", *node.Key, r)
			if err != nil {
				return nil, err
			}
		}
		_, err = dal.Q.Chain.WithContext(c).Where(dal.Chain.ID.Eq(r.ID)).Delete(&entity.Chain{})
		if err != nil {
			return nil, err
		}
		if _, err := dal.UserRoleChainMapping.WithContext(c).Where(dal.UserRoleChainMapping.UserID.Eq(ctx.GetInt32(common.HeaderUserKey)), dal.UserRoleChainMapping.ChainID.Eq(int32(r.ID))).Delete(&entity.UserRoleChainMapping{}); err != nil {
			return nil, err
		}
		return &resp.DelChainResp{ChainId: int64(chainM.ChainID)}, nil
	}
	chain, err := dal.Q.Chain.WithContext(c).Where(dal.Chain.ID.Eq(r.ID)).Preload(dal.Chain.Node).First()
	if err != nil {
		return nil, err
	}
	cnt, err := dal.Q.Chain.WithContext(c).Where(dal.Chain.NodeID.Eq(chain.NodeID), dal.Chain.Port.Eq(*chain.Port)).Count()
	if err != nil {
		return nil, err
	}
	// may ipv4 ipv6 all use one port, so only count == 1, delete port
	if cnt == 1 && chain.Node.Key != nil {
		r.Port = *chain.Port
		_, err = client.DoRequest(fmt.Sprintf("%s:%d", *chain.Node.IP, *chain.Node.Port), "chain", "DeleteChain", *chain.Node.Key, r)
		if err != nil {
			return nil, err
		}
	}
	_, err = dal.Q.Chain.WithContext(c).Where(dal.Chain.ID.Eq(r.ID)).Delete(&entity.Chain{})
	if err != nil {
		return nil, err
	}

	return &resp.DelChainResp{ChainId: chain.ID}, nil
}

func listChain(c context.Context, ctx *app.RequestContext) (*resp.ListChainResp, error) {
	req := req.ListChainReq{}
	if err := ctx.BindAndValidate(&req); err != nil {
		return nil, err
	}

	if config.GetRole() == string(common.Agent) {
		return &resp.ListChainResp{}, nil
	}

	var chainIds []int64
	if err := dal.UserRoleChainMapping.WithContext(c).Select(dal.UserRoleChainMapping.ChainID).
		Where(dal.UserRoleChainMapping.UserID.Eq(ctx.GetInt32(common.HeaderUserKey))).
		Or(dal.UserRoleChainMapping.RoleCode.Eq(ctx.GetString(common.HeaderRoleKey))).Scan(&chainIds); err != nil {
		return nil, err
	}

	var q []gen.Condition
	if ctx.GetString(common.HeaderRoleKey) != "SUPER_ADMIN" {
		q = append(q, dal.Chain.ID.In(chainIds...))
	}
	if req.Protocol != "" {
		q = append(q, dal.Chain.Protocol.Eq(req.Protocol))
	}
	if req.ChainName != "" {
		q = append(q, dal.Chain.ChainName.Like("%"+req.ChainName+"%"))
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
	chains, cnt, err := dal.Chain.WithContext(c).
		Where(q...).Preload(dal.Chain.Node).Order(dal.Chain.UpdatedAt.Desc()).FindByPage(int((req.PageNo-1)*req.PageSize), int(req.PageSize))
	if err != nil {
		return nil, err
	}

	return &resp.ListChainResp{Chains: chains, TotalCount: cnt}, nil
}

func editChain(ctx context.Context, c *app.RequestContext) (*resp.EditChainResp, error) {
	r := req.EditChainReq{}
	if err := c.BindAndValidate(&r); err != nil {
		return nil, err
	}

	if config.GetRole() == string(common.Agent) {
		return &resp.EditChainResp{}, nil
	}

	_, err := dal.Chain.WithContext(ctx).Where(dal.Chain.ID.Eq(r.ID)).Update(dal.Chain.ChainName, r.ChainName)
	if err != nil {
		return nil, err
	}
	return &resp.EditChainResp{}, nil
}
