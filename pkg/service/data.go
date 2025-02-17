package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gorm.io/gorm/clause"

	"github.com/Mr-LvGJ/stander/pkg/common"
	"github.com/Mr-LvGJ/stander/pkg/model/dal"
	"github.com/Mr-LvGJ/stander/pkg/model/entity"
	error2 "github.com/Mr-LvGJ/stander/pkg/service/error"
	"github.com/Mr-LvGJ/stander/pkg/service/req"
	"github.com/Mr-LvGJ/stander/pkg/service/resp"
	"github.com/Mr-LvGJ/stander/pkg/utils/cron"
)

var userPlanUsageMap = sync.Map{}

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

	rate := node.Rate
	var (
		rule *entity.Rule
	)
	rule, err = dal.Rule.WithContext(ctx).
		Where(dal.Rule.NodeID.Eq(node.ID), dal.Rule.ListenPort.Eq(r.Port)).
		First()
	if err != nil {
		return nil, err
	}

	if *rule.ChainID != 0 {
		chain, err := dal.Chain.WithContext(ctx).Preload(dal.Chain.Node).Where(dal.Chain.ID.Eq(*rule.ChainID)).First()
		if err != nil {
			return nil, err
		}
		rate += chain.Node.Rate
	}
	addTraffic := int64(float32(r.Traffic) * rate)
	hlog.Infof("report traffic, ruleName: %s, rate: %.2f, addtraffic: %d", *rule.RuleName, rate, addTraffic)
	_, err = dal.Rule.WithContext(ctx).
		Where(dal.Rule.NodeID.Eq(node.ID), dal.Rule.ListenPort.Eq(r.Port)).
		UpdateColumn(dal.Rule.Traffic, dal.Rule.Traffic.Add(addTraffic))
	if err != nil {
		return nil, err
	}

	today, _ := time.Parse(time.DateOnly, time.Now().Format(time.DateOnly))
	if err := dal.UserDailyTraffic.WithContext(ctx).Clauses(
		clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]interface{}{"total_traffic": dal.UserDailyTraffic.TotalTraffic.Add(addTraffic)}),
		},
	).Create(&entity.UserDailyTraffic{
		Date:         today,
		TotalTraffic: r.Traffic,
		UserID:       *rule.UserID,
	}); err != nil {
		return nil, err
	}
	return &resp.ReportNetworkTrafficResp{}, nil
}

func calculateUserPeriodTrafficUsage(ctx context.Context) error {
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			users, err := dal.User.WithContext(ctx).Preload(dal.User.TrafficPlan).Find()
			if err != nil {
				hlog.Errorf("calculateUserPeriodTrafficUsage get user from db failed, err: %v", err)
				return
			}
			for _, user := range users {
				var expirationTime time.Time
				if user.ExpirationTime != nil {
					expirationTime = *user.ExpirationTime
				}
				v, ok := userPlanUsageMap.Load(*user.ID)
				if !ok {
					hlog.Infof("UserPlanUsage: %s, Taffic: %s GB, %v",
						fmt.Sprintf("%8s", *user.Username),
						fmt.Sprintf("%10s", fmt.Sprintf("%d/%d", 0, user.TrafficPlan.TotalTraffic/1024/1024/1024)),
						expirationTime)
					continue
				}
				used := v.(int64)
				hlog.Infof("UserPlanUsage: %s, Taffic: %s GB, %v",
					fmt.Sprintf("%8s", *user.Username),
					fmt.Sprintf("%10s", fmt.Sprintf("%d/%d", used/1024/1024/1024, user.TrafficPlan.TotalTraffic/1024/1024/1024)),
					expirationTime)
			}
			//userPlanUsageMap.Range(func(key, value interface{}) bool {
			//	v := value.(int64)
			//	hlog.Debugf("userPlanUsageMap, key: %v, value: %v", key, v/1024/1024/1024)
			//	return true
			//})
		}
	}()
	f := func(ctx context.Context) error {
		// 1. 计算周期
		users, err := dal.User.WithContext(ctx).Preload(dal.User.TrafficPlan).Find()
		if err != nil {
			return err
		}
		// 2. 计算用户当前周期用量
		for _, user := range users {
			//hlog.Infof("user_id: %d, user_name: %s, planId: %d", user.ID, *user.Username, user.PlanID)
			if user.ExpirationTime == nil || user.TrafficPlan.Period == nil {
				continue
			}
			to := *user.ExpirationTime
			from := *user.ExpirationTime
			switch entity.PlanPeriod(*user.TrafficPlan.Period) {
			case entity.Month:
				from = from.AddDate(0, -1, -1)
			case entity.Quarter:
				from = from.AddDate(0, -3, -1)
			case entity.HalfYear:
				from = from.AddDate(0, -6, -1)
			case entity.Year:
				from = from.AddDate(-1, 0, -1)
			}

			var periodTotalTraffic int64
			if err := dal.UserDailyTraffic.WithContext(ctx).
				Select(dal.UserDailyTraffic.TotalTraffic.Sum().IfNull(0)).
				Where(
					dal.UserDailyTraffic.UserID.Eq(*user.ID),
					dal.UserDailyTraffic.Date.Gte(from),
					dal.UserDailyTraffic.Date.Lte(to),
				).Scan(&periodTotalTraffic); err != nil {
				return err
			}
			//hlog.Infof("calculateUserPeriodTrafficUsage, username: %v, from: %v, to: %v", *user.Username, from, to)
			userPlanUsageMap.Store(*user.ID, periodTotalTraffic)
		}
		return nil
	}
	c := cron.New(30 * time.Second)
	if err := c.Do(ctx, f); err != nil {
		return err
	}
	return nil
}
