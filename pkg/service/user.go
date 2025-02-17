package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gen"

	"github.com/Mr-LvGJ/stander/pkg/model/dal"
	error2 "github.com/Mr-LvGJ/stander/pkg/service/error"
	"github.com/Mr-LvGJ/stander/pkg/service/req"
	"github.com/Mr-LvGJ/stander/pkg/service/resp"
)

func UserSrv(c context.Context, ctx *app.RequestContext) {
	action := ctx.Query("Action")
	switch action {
	case "GetUserPlanInfo":
		resp, err := getUserPlanInfo(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	case "ListUsers":
		resp, err := listUsers(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	case "EditUser":
		resp, err := editUser(c, ctx)
		error2.WriteResponse(ctx, err, resp)

	default:
		error2.WriteResponse(ctx, errors.New("action not found"), nil)
	}
}

func getUserPlanInfo(ctx context.Context, c *app.RequestContext) (*resp.GetUserPlanInfoResp, error) {
	r := req.GetUserPlanInfoReq{}
	if err := c.BindAndValidate(&r); err != nil {
		return nil, err
	}
	user, err := dal.User.WithContext(ctx).Where(dal.User.ID.Eq(r.UserId)).Preload(dal.User.TrafficPlan).First()
	if err != nil {
		return nil, err
	}
	dfs, err := dal.UserDailyTraffic.WithContext(ctx).Where(dal.UserDailyTraffic.UserID.Eq(*user.ID)).
		Order(dal.UserDailyTraffic.Date.Asc()).Limit(30).Find()
	if err != nil {
		return nil, err
	}
	expireTime := time.Now()
	usedTraffic := int64(0)
	plan := "测试套餐"
	if user.ExpirationTime != nil {
		expireTime = *user.ExpirationTime
	}
	if user.TrafficPlan.PlanName != nil {
		plan = *user.TrafficPlan.PlanName
	}
	value, ok := userPlanUsageMap.Load(*user.ID)
	if ok {
		usedTraffic = value.(int64) / 1024 / 1024 / 1024
	}

	var dailyTraffics []resp.DailyTraffic
	for _, df := range dfs {
		dailyTraffics = append(dailyTraffics, resp.DailyTraffic{
			Date:    df.Date,
			Traffic: int32(df.TotalTraffic / 1024 / 1024 / 1024),
		})

	}
	return &resp.GetUserPlanInfoResp{
		Username:       *user.Username,
		ExpirationTime: expireTime,
		PlanTraffic:    int32(user.TrafficPlan.TotalTraffic / 1024 / 1024 / 1024),
		UsedTraffic:    int32(usedTraffic),
		PlanName:       plan,
		DailyTraffics:  dailyTraffics,
	}, nil
}

func listUsers(ctx context.Context, c *app.RequestContext) (*resp.ListUsersResp, error) {
	r := req.ListUsersReq{}
	if err := c.BindAndValidate(&r); err != nil {
		return nil, err
	}
	var q []gen.Condition

	if r.Username != "" {
		q = append(q, dal.User.Username.Like(fmt.Sprintf("%%%s%%", r.Username)))
	}

	od := dal.User.ExpirationTime.Asc()

	if r.OrderBy != "" {
		name, ok := dal.User.GetFieldByName(r.OrderBy)
		if ok {
			od = name.Desc()
			if r.Asc {
				od = name.Asc()
			}
		}
	}

	users, count, err := dal.User.WithContext(ctx).
		Where(q...).
		Order(od).
		Preload(dal.User.TrafficPlan).
		FindByPage(r.PageSize*(r.PageNo-1), r.PageSize)
	if err != nil {
		return nil, err
	}
	var userTos []*resp.UserTo
	for _, user := range users {
		usedTraffic := int64(0)
		value, ok := userPlanUsageMap.Load(*user.ID)
		if ok {
			usedTraffic = value.(int64) / 1024 / 1024 / 1024
		}
		userTos = append(userTos, &resp.UserTo{
			User:        user,
			UsedTraffic: usedTraffic,
		})
	}
	return &resp.ListUsersResp{
		PageSize:   r.PageSize,
		PageNumber: r.PageNo,
		TotalCount: count,
		Users:      userTos,
	}, nil
}

func editUser(ctx context.Context, c *app.RequestContext) (*resp.EmptyResp, error) {
	r := req.EditUserReq{}
	if err := c.BindAndValidate(&r); err != nil {
		return nil, err
	}
	updatedFields := make(map[string]any)
	if r.ExpirationTime != nil {
		updatedFields["expiration_time"] = r.ExpirationTime
	}
	_, err := dal.User.WithContext(ctx).Where(dal.User.ID.Eq(r.ID)).Updates(updatedFields)
	if err != nil {
		return nil, err
	}

	return &resp.EmptyResp{}, nil

}
