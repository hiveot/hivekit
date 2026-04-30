package wotcli

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
)

const WotCLIModuleType = "wotcli"

type WotCLI struct {
	modules.HiveModuleBase
}

func NewWotCLI() *WotCLI {
	cl := &WotCLI{}
	return cl
}

func NewWotCLIFactory(f factory.IModuleFactory) modules.IHiveModule {
	cl := NewWotCLI()
	return cl
}
