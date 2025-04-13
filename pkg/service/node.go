package service

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/google/uuid"
	"github.com/thoas/go-funk"
	"gorm.io/gen"
	"gorm.io/gorm"

	"github.com/Mr-LvGJ/stander/pkg/common"
	"github.com/Mr-LvGJ/stander/pkg/config"
	"github.com/Mr-LvGJ/stander/pkg/model/dal"
	"github.com/Mr-LvGJ/stander/pkg/model/entity"
	error2 "github.com/Mr-LvGJ/stander/pkg/service/error"
	"github.com/Mr-LvGJ/stander/pkg/service/req"
	"github.com/Mr-LvGJ/stander/pkg/service/resp"
	"github.com/Mr-LvGJ/stander/pkg/utils"
)

func NodeSrv(c context.Context, ctx *app.RequestContext) {
	action := ctx.Query("Action")
	switch action {
	case "AddNode":
		resp, err := addNode(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	case "RegisterNode":
		resp, err := registerNode(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	case "ListNodes":
		resp, err := listNode(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	case "GetNodePermissions":
		resp, err := getNodePermissions(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	case "ListNodeChainRelationShips":
		resp, err := listNodeChainRelationShips(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	case "DeleteNode":
		resp, err := delNode(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	case "EditNode":
		resp, err := editNode(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	default:
		error2.WriteResponse(ctx, errors.New("action not found"), nil)
	}
}

func checkUserNodePermission(ctx context.Context, c *app.RequestContext, nodeIds ...int64) error {
	if c.GetString(common.HeaderRoleKey) == common.SUPER_ADMIN {
		return nil
	}
	var availableNodeIds []int64
	if err := dal.UserRoleNodeMapping.WithContext(ctx).Select(dal.UserRoleNodeMapping.NodeID).
		Where(dal.UserRoleNodeMapping.UserID.Eq(c.GetInt32(common.HeaderUserKey))).
		Scan(&availableNodeIds); err != nil {
		return err
	}

	for _, id := range nodeIds {
		if !funk.ContainsInt64(availableNodeIds, id) {
			return permissionDenyError
		}
	}
	return nil
}

func addNode(c context.Context, ctx *app.RequestContext) (*resp.AddNodeResp, error) {
	r := req.AddNodeReq{}
	if err := ctx.BindAndValidate(&r); err != nil {
		return nil, err
	}
	uid := uuid.New().String()
	if ctx.GetString(common.HeaderRoleKey) != "SUPER_ADMIN" {
		if err := dal.Q.Node.WithContext(c).Create(&entity.Node{
			NodeName: &r.NodeName,
			Key:      &uid,
			NodeType: &r.NodeType,
			Rate:     r.Rate,
		}); err != nil {
			return nil, err
		}
		node, _ := dal.Node.WithContext(c).Where(dal.Node.Key.Eq(uid)).First()
		userId := ctx.GetInt32(common.HeaderUserKey)
		if err := dal.UserRoleNodeMapping.WithContext(c).Create(&entity.UserRoleNodeMapping{
			UserID: &userId,
			NodeID: int32(node.ID),
		}); err != nil {
			return nil, err
		}
	} else {
		if err := dal.Q.Node.WithContext(c).Create(&entity.Node{
			NodeName: &r.NodeName,
			Key:      &uid,
			NodeType: &r.NodeType,
			Rate:     r.Rate,
		}); err != nil {
			return nil, err
		}
		node, _ := dal.Node.WithContext(c).Where(dal.Node.Key.Eq(uid)).First()
		roleCode := ctx.GetString(common.HeaderRoleKey)
		if err := dal.UserRoleNodeMapping.WithContext(c).Create(&entity.UserRoleNodeMapping{
			RoleCode: &roleCode,
			NodeID:   int32(node.ID),
		}); err != nil {
			return nil, err
		}
	}

	return &resp.AddNodeResp{Key: uid}, nil
}

func delNode(c context.Context, ctx *app.RequestContext) (*resp.DelNodeResp, error) {
	r := req.DelNodeReq{}
	if err := ctx.BindAndValidate(&r); err != nil {
		return nil, err
	}
	if ctx.GetString(common.HeaderRoleKey) != "SUPER_ADMIN" {
		nodeM, err := dal.UserRoleNodeMapping.WithContext(c).
			Where(dal.UserRoleNodeMapping.UserID.Eq(ctx.GetInt32(common.HeaderUserKey)), dal.UserRoleNodeMapping.NodeID.Eq(int32(r.ID))).
			Preload(dal.UserRoleNodeMapping.Node).First()
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, permissionDenyError
			}
			return nil, err
		}
		node := nodeM.Node
		_, err = dal.Q.Node.WithContext(c).Where(dal.Node.ID.Eq(node.ID)).Delete(&entity.Node{})
		if err != nil {
			return nil, err
		}
		if _, err := dal.UserRoleNodeMapping.WithContext(c).Where(dal.UserRoleNodeMapping.NodeID.Eq(int32(node.ID))).Delete(&entity.UserRoleNodeMapping{}); err != nil {
			return nil, err
		}
		return &resp.DelNodeResp{ID: r.ID}, nil
	}

	_, err := dal.Q.Node.WithContext(c).Where(dal.Node.ID.Eq(r.ID)).Delete(&entity.Node{})
	if err != nil {
		return nil, err
	}

	if _, err := dal.UserRoleNodeMapping.WithContext(c).Where(dal.UserRoleNodeMapping.NodeID.Eq(int32(r.ID))).Delete(&entity.UserRoleNodeMapping{}); err != nil {
		return nil, err
	}

	return &resp.DelNodeResp{ID: r.ID}, nil
}

func registerNode(c context.Context, ctx *app.RequestContext) (*resp.RegisterNodeResp, error) {
	r := req.RegisterNodeReq{}
	if err := ctx.BindAndValidate(&r); err != nil {
		return nil, err
	}
	res := &resp.RegisterNodeResp{
		Chains: []*resp.ChainVO{},
		Rules:  []*resp.RuleVO{},
	}
	key := string(ctx.GetHeader(common.KeyHeader))
	node, err := dal.Q.Node.WithContext(c).Where(dal.Node.Key.Eq(key)).First()
	if err != nil {
		return nil, err
	}
	clientIP := r.Ipv4
	port := r.Port
	ipv4 := r.Ipv4
	ipv6 := r.Ipv6
	managerIp := r.ManagerIp

	if r.PreferIpv6 {
		clientIP = ipv6
	}

	hlog.Infof("Node: RegisterNode, ClientIP: %s, Port: %d", clientIP, port)
	if r.Port == 0 {
		port = int32(8123)
	}
	if clientIP == "" {
		clientIP = ctx.ClientIP()
	}

	if managerIp == "" {
		managerIp = clientIP
	}
	rules, err := dal.Rule.WithContext(c).Where(dal.Rule.NodeID.Eq(node.ID)).Find()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	chains, err := dal.Chain.WithContext(c).Where(dal.Chain.NodeID.Eq(node.ID)).Find()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	for _, chain := range chains {
		res.Chains = append(res.Chains, &resp.ChainVO{
			ChainType: *chain.Protocol,
			Port:      *chain.Port,
		})
	}
	for _, rule := range rules {
		ruleVo := &resp.RuleVO{
			ListenPort: *rule.ListenPort,
			RemoteAddr: *rule.RemoteAddr,
			ChainType:  *rule.Protocol,
		}
		chain, err := dal.Chain.WithContext(c).Where(dal.Chain.ID.Eq(*rule.ChainID)).First()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if chain != nil {
			ruleVo.ChainAddr = utils.GenIpAndPort(*chain.IP, *chain.Port)
		}
		res.Rules = append(res.Rules, ruleVo)
	}
	_, err = dal.Q.Node.WithContext(c).Where(dal.Node.Key.Eq(key)).Updates(entity.Node{
		IP:        &clientIP,
		Port:      &port,
		Ipv4:      &ipv4,
		Ipv6:      &ipv6,
		ManagerIP: managerIp,
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func listNode(c context.Context, ctx *app.RequestContext) (*resp.ListNodeResp, error) {
	r := req.ListNodeReq{}
	if err := ctx.BindAndValidate(&r); err != nil {
		return nil, err
	}
	if config.GetRole() == string(common.Agent) {
		return &resp.ListNodeResp{}, nil
	}

	var nodeIds []int64
	if r.Scene == "" || ctx.GetString(common.HeaderRoleKey) == common.SUPER_ADMIN {
		if err := dal.UserRoleNodeMapping.WithContext(c).Select(dal.UserRoleNodeMapping.NodeID).
			Where(dal.UserRoleNodeMapping.UserID.Eq(ctx.GetInt32(common.HeaderUserKey))).
			Or(dal.UserRoleNodeMapping.RoleCode.Eq(ctx.GetString(common.HeaderRoleKey))).Scan(&nodeIds); err != nil {
			return nil, err
		}
	} else if r.Scene == req.AddChainScene {
		if err := dal.UserRoleNodeMapping.WithContext(c).Select(dal.UserRoleNodeMapping.NodeID).
			Where(dal.UserRoleNodeMapping.UserID.Eq(ctx.GetInt32(common.HeaderUserKey))).
			Scan(&nodeIds); err != nil {
			return nil, err
		}
	}

	var q []gen.Condition
	if ctx.GetString(common.HeaderRoleKey) != "SUPER_ADMIN" {
		q = append(q, dal.Node.ID.In(nodeIds...))
	}
	if r.NodeType != "" {
		q = append(q, dal.Node.NodeType.Eq(r.NodeType))
	}
	if r.NodeName != "" {
		q = append(q, dal.Node.NodeName.Like("%"+r.NodeName+"%"))
	}
	if r.PageSize == 0 {
		r.PageSize = 10
	}
	if r.PageNo == 0 {
		r.PageNo = 1
	}
	if r.PageNo == -1 {
		r.PageSize = 1000
	}
	nodes, cnt, err := dal.Node.WithContext(c).
		Where(q...).Order(dal.Node.UpdatedAt.Desc()).
		FindByPage(int((r.PageNo-1)*r.PageSize), int(r.PageSize))
	if err != nil {
		return nil, err
	}

	return &resp.ListNodeResp{Nodes: nodes, TotalCount: cnt}, nil
}

func editNode(c context.Context, ctx *app.RequestContext) (*resp.EditNodeResp, error) {
	r := req.EditNodeReq{}
	if err := ctx.BindAndValidate(&r); err != nil {
		return nil, err
	}
	if config.GetRole() == string(common.Agent) {
		return &resp.EditNodeResp{}, nil
	}

	if err := checkUserNodePermission(c, ctx, r.ID); err != nil {
		return nil, err
	}

	updateFields := make(map[string]any)
	if r.NodeName != "" {
		updateFields["node_name"] = r.NodeName
	}
	if r.Rate >= 0 {
		updateFields["rate"] = r.Rate
	}

	_, err := dal.Node.WithContext(c).Where(dal.Node.ID.Eq(r.ID)).Updates(updateFields)
	if err != nil {
		return nil, err
	}
	return &resp.EditNodeResp{}, nil
}

func listNodeChainRelationShips(ctx context.Context, c *app.RequestContext) (*resp.ListNodeChainRelationShipsResp, error) {
	r := req.ListNodeChainRelationShipsReq{}
	if err := c.BindAndValidate(&r); err != nil {
		return nil, err
	}

	node, err := dal.Node.WithContext(ctx).Where(dal.Node.ID.Eq(r.NodeId)).First()
	if err != nil {
		return nil, err
	}
	if node.Iepl != 0 {
		return &resp.ListNodeChainRelationShipsResp{}, err
	}

	var availableChainIds []int64
	if err := dal.UserRoleChainMapping.WithContext(ctx).Select(dal.UserRoleChainMapping.ChainID).
		Where(dal.UserRoleChainMapping.UserID.Eq(c.GetInt32(common.HeaderUserKey))).
		Or(dal.UserRoleChainMapping.RoleCode.Eq(c.GetString(common.HeaderRoleKey))).Scan(&availableChainIds); err != nil {
		return nil, err
	}

	var nodeAvailableChainIds []int64
	if err := dal.NodeChainMapping.WithContext(ctx).Select(dal.NodeChainMapping.ChainID).Where(
		dal.NodeChainMapping.NodeID.Eq(r.NodeId)).Scan(&nodeAvailableChainIds); err != nil {
		return nil, err
	}

	var userSelfChainIds []int64
	if err := dal.UserRoleChainMapping.WithContext(ctx).Select(dal.UserRoleChainMapping.ChainID).
		Where(dal.UserRoleChainMapping.UserID.Eq(c.GetInt32(common.HeaderUserKey))).
		Scan(&userSelfChainIds); err != nil {
		return nil, err
	}

	chains, err := dal.Chain.WithContext(ctx).Where(dal.Chain.ID.In(availableChainIds...),
		dal.Chain.ID.In(nodeAvailableChainIds...)).
		Or(dal.Chain.ID.In(userSelfChainIds...)).Find()
	if err != nil {
		return nil, err
	}

	return &resp.ListNodeChainRelationShipsResp{
		Chains: chains,
	}, nil
}

func getNodePermissions(ctx context.Context, c *app.RequestContext) ([]int64, error) {
	r := req.EmptyReq{}
	if err := c.BindAndValidate(&r); err != nil {
		return nil, err
	}
	var availableNodeIds []int64
	if c.GetString(common.HeaderRoleKey) == common.SUPER_ADMIN {
		if err := dal.Node.WithContext(ctx).Select(dal.Node.ID).Scan(&availableNodeIds); err != nil {
			return nil, err
		}
	} else {
		if err := dal.UserRoleNodeMapping.WithContext(ctx).Select(dal.UserRoleNodeMapping.NodeID).
			Where(dal.UserRoleNodeMapping.UserID.Eq(c.GetInt32(common.HeaderUserKey))).
			Scan(&availableNodeIds); err != nil {
			return nil, err
		}
	}
	return availableNodeIds, nil
}
