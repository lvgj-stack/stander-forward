package service

import (
	"context"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func Setup(ctx context.Context) {
	go func() {
		err := calculateUserPeriodTrafficUsage(ctx)
		if err != nil {
			hlog.Errorf("calculateUserPeriodTrafficUsage failed, err: %v", err)
		}
	}()
}
