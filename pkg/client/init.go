package client

import (
	"context"

	"github.com/Mr-LvGJ/stander/pkg/config"
)

func Init() {
	if config.GetAgentConfig().EnableGost {
		InitGostCli(context.Background())
	}
}
