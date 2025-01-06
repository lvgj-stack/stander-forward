package service

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/Mr-LvGJ/stander/pkg/common"
	"github.com/Mr-LvGJ/stander/pkg/model/dal"
	error2 "github.com/Mr-LvGJ/stander/pkg/service/error"
	"github.com/Mr-LvGJ/stander/pkg/service/req"
	"github.com/Mr-LvGJ/stander/pkg/service/resp"
)

func DataSrv(c context.Context, ctx *app.RequestContext) {
	action := ctx.Query("Action")
	switch action {
	case "ReportNetworkTraffic":
		resp, err := reportNetworkTraffic(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	default:
		error2.WriteResponse(ctx, errors.New("action not found"), nil)
	}
}

func reportNetworkTraffic(ctx context.Context, c *app.RequestContext) (*resp.ReportNetworkTrafficResp, error) {
	r := req.ReportNetworkTrafficReq{}
	if err := c.BindAndValidate(&r); err != nil {
		return nil, err
	}
	k := string(c.GetHeader(common.KeyHeader))
	if k == "" {
		return nil, errors.New("header key not found")
	}
	node, err := dal.Node.WithContext(ctx).Where(dal.Node.Key.Eq(k)).First()
	if err != nil {
		return nil, err
	}
	_, err = dal.Rule.WithContext(ctx).
		Where(dal.Rule.NodeID.Eq(node.ID), dal.Rule.ListenPort.Eq(r.Port)).
		UpdateColumn(dal.Rule.Traffic, dal.Rule.Traffic.Add(r.Traffic))
	if err != nil {
		return nil, err
	}
	return &resp.ReportNetworkTrafficResp{}, nil
}
