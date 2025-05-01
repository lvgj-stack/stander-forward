package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"github.com/Mr-LvGJ/stander/pkg/common"
	"github.com/Mr-LvGJ/stander/pkg/config"
	"github.com/Mr-LvGJ/stander/pkg/service"
)

// req.Header.Set("X-User-Id", strconv.Itoa(ctx.GetInt("uid")))
// req.Header.Set("X-Role-Id", ctx.GetString("roleId"))
func InitControllerRoute(h *server.Hertz) {
	api := h.Group("/api/v1")
	api.Use(func(c context.Context, ctx *app.RequestContext) {
		roleId := string(ctx.GetHeader(common.HeaderRoleKey))
		uid, _ := strconv.Atoi(string(ctx.GetHeader(common.HeaderUserKey)))
		ctx.Set(common.HeaderRoleKey, roleId)
		ctx.Set(common.HeaderUserKey, int32(uid))
		c = context.WithValue(c, common.HeaderRoleKey, roleId)
		c = context.WithValue(c, common.HeaderUserKey, int32(uid))
	})
	api.POST("rule", service.RuleSrv)
	api.POST("chain", service.ChainSrv)
	api.POST("chain-group", service.ChainGroupSrv)
	api.POST("node", service.NodeSrv)
	api.POST("data", service.DataSrv)
	api.POST("user", service.UserSrv)
}

func InitAgentRoute(h *server.Hertz) {
	api := h.Group("/api/v1")
	// for gost
	api.POST("data", service.DataSrv)
	api.Use(func(c context.Context, ctx *app.RequestContext) {
		k := string(ctx.GetHeader(common.KeyHeader))
		if k != config.GetKey() {
			ctx.JSON(http.StatusForbidden, map[string]any{
				"Error": "request forbidden",
			})
		}
	})
	api.POST("rule", service.RuleSrv)
	api.POST("chain", service.ChainSrv)
	api.POST("node", service.NodeSrv)
}
