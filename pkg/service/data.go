package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gorm.io/gorm/clause"

	"github.com/Mr-LvGJ/stander/pkg/client"
	"github.com/Mr-LvGJ/stander/pkg/common"
	"github.com/Mr-LvGJ/stander/pkg/config"
	"github.com/Mr-LvGJ/stander/pkg/model/dal"
	"github.com/Mr-LvGJ/stander/pkg/model/entity"
	error2 "github.com/Mr-LvGJ/stander/pkg/service/error"
	"github.com/Mr-LvGJ/stander/pkg/service/req"
	"github.com/Mr-LvGJ/stander/pkg/service/resp"
	"github.com/Mr-LvGJ/stander/pkg/utils/cron"
)

var userPlanUsageMap = sync.Map{}
var nodeTrafficMap = sync.Map{}

func DataSrv(c context.Context, ctx *app.RequestContext) {
	action := ctx.Query("Action")
	switch action {
	case "ReportNetworkTraffic":
		resp, err := reportNetworkTraffic(c, ctx)
		error2.WriteResponse(ctx, err, resp)
	case "ObserverNetworkTraffic":
		_, _ = observerNetworkTraffic(c, ctx)
		ctx.JSON(200, map[string]bool{"ok": true})
	default:
		error2.WriteResponse(ctx, errors.New("action not found"), nil)
	}
}

func observerNetworkTraffic(ctx context.Context, c *app.RequestContext) (*resp.ReportNetworkTrafficResp, error) {
	r := req.ObserverNetworkTrafficReq{}
	if err := c.BindAndValidate(&r); err != nil {
		hlog.Errorf("[ObserverNetworkTraffic] err: %v", err)
		return nil, err
	}
	hlog.Infof("[ObserverNetworkTraffic] req: %v", r)

	cfg := config.GetAgentConfig()
	for _, event := range r.Events {
		if strings.Contains(event.Service, "udp") ||
			strings.Contains(event.Service, "chain") {
			continue
		}
		//var preTraffic int64
		//v, ok := nodeTrafficMap.Load(event.Service)
		//if ok {
		//	preTraffic = v.(int64)
		//}
		ss := strings.Split(event.Service, "-")
		port, _ := strconv.Atoi(ss[len(ss)-1])
		_, err := client.DoRequest(cfg.ControllerAddr, "data", "ReportNetworkTraffic", cfg.NodeKey, &req.ReportNetworkTrafficReq{
			Port:    int32(port),
			Traffic: event.Stats.InputBytes + event.Stats.OutputBytes,
		})
		if err != nil {
			hlog.Errorf("[ObserverNetworkTraffic] ReportNetworkTraffic err: %v", err)
			return nil, err
		}
		//nodeTrafficMap.Store(event.Service, event.Stats.InputBytes+event.Stats.OutputBytes)
	}
	return &resp.ReportNetworkTrafficResp{}, nil
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
	userResetChan := make(chan *entity.User, 10)

	go func() {
		for {
			select {
			case user := <-userResetChan:
				nextResetTime := time.Now()
				switch entity.PlanPeriod(*user.TrafficPlan.Period) {
				case entity.Month:
					nextResetTime = nextResetTime.AddDate(0, 1, 0)
				case entity.Quarter:
					nextResetTime = nextResetTime.AddDate(0, 3, 0)
				case entity.HalfYear:
					nextResetTime = nextResetTime.AddDate(0, 6, 0)
				case entity.Year:
					nextResetTime = nextResetTime.AddDate(1, 0, 0)
				}

				if user.ExpirationTime.Add(5 * time.Hour).After(nextResetTime) {
					_, err := dal.User.WithContext(ctx).Where(dal.User.ID.Eq(*user.ID)).Update(dal.User.ResetTrafficTime, nextResetTime)
					if err != nil {
						hlog.Errorf("failed to refresh user reset time, user: %v, err: %v", user, err)
						continue
					}
				} else {
					hlog.Warnf("user expired! user: %s", *user.Username)
				}
			}
		}
	}()
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			users, err := dal.User.WithContext(ctx).Preload(dal.User.TrafficPlan).Find()
			if err != nil {
				hlog.Errorf("calculateUserPeriodTrafficUsage get user from db failed, err: %v", err)
				return
			}
			for _, user := range users {
				var resetTime time.Time
				if user.ResetTrafficTime != nil {
					resetTime = *user.ResetTrafficTime
				}
				v, ok := userPlanUsageMap.Load(*user.ID)
				if !ok {
					hlog.Infof("UserPlanUsage: %s, Taffic: %s GB, %v",
						fmt.Sprintf("%8s", *user.Username),
						fmt.Sprintf("%10s", fmt.Sprintf("%d/%d", 0, user.TrafficPlan.TotalTraffic/1024/1024/1024)),
						resetTime)
					continue
				}
				used := v.(int64)
				hlog.Infof("UserPlanUsage: %s, Taffic: %s GB, %v",
					fmt.Sprintf("%8s", *user.Username),
					fmt.Sprintf("%10s", fmt.Sprintf("%d/%d", used/1024/1024/1024, user.TrafficPlan.TotalTraffic/1024/1024/1024)),
					resetTime)
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
			if user.ResetTrafficTime == nil || user.TrafficPlan.Period == nil {
				continue
			}
			to := *user.ResetTrafficTime
			from := *user.ResetTrafficTime

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
			dailyTraffics, err := dal.UserDailyTraffic.WithContext(ctx).
				Select(dal.UserDailyTraffic.TotalTraffic).
				Where(
					dal.UserDailyTraffic.UserID.Eq(*user.ID),
					dal.UserDailyTraffic.Date.Gte(from),
					dal.UserDailyTraffic.Date.Lte(to),
				).Find()
			if err != nil {
				return err
			}
			for _, traffic := range dailyTraffics {
				periodTotalTraffic += traffic.TotalTraffic
			}

			// 当前时间超过了重置时间 or 使用流量超量了
			if time.Now().After(to) || periodTotalTraffic > user.TrafficPlan.TotalTraffic {
				userResetChan <- user
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
