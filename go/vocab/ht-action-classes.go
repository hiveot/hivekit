// Package vocab with HiveOT and WoT vocabulary names for TD Things, properties, events and actions
package vocab

// type: ActionClasses
// namespace: hiveot
const (
	ActionDimmer              = "hiveot:action:dimmer"
	ActionDimmerDecrement     = "hiveot:action:dimmer:decrement"
	ActionDimmerIncrement     = "hiveot:action:dimmer:increment"
	ActionDimmerSet           = "hiveot:action:dimmer:set"
	ActionMedia               = "hiveot:action:media"
	ActionMediaMute           = "hiveot:action:media:mute"
	ActionMediaNext           = "hiveot:action:media:next"
	ActionMediaPause          = "hiveot:action:media:pause"
	ActionMediaPlay           = "hiveot:action:media:play"
	ActionMediaPrevious       = "hiveot:action:media:previous"
	ActionMediaUnmute         = "hiveot:action:media:unmute"
	ActionMediaVolume         = "hiveot:action:media:volume"
	ActionMediaVolumeDecrease = "hiveot:action:media:volume:decrease"
	ActionMediaVolumeIncrease = "hiveot:action:media:volume:increase"
	ActionSwitch              = "hiveot:action:switch"
	ActionSwitchOnOff         = "hiveot:action:switch:onoff"
	ActionSwitchToggle        = "hiveot:action:switch:toggle"
	ActionThingDisable        = "hiveot:action:thing:disable"
	ActionThingEnable         = "hiveot:action:thing:enable"
	ActionThingStart          = "hiveot:action:thing:start"
	ActionThingStop           = "hiveot:action:thing:stop"
	ActionValveClose          = "hiveot:action:valve:close"
	ActionValveOpen           = "hiveot:action:valve:open"
)

// end of ActionClasses

// ActionClassesMap maps @type to symbol, title and description
var ActionClassesMap = map[string]struct {
	Symbol      string
	Title       string
	Description string
}{
	ActionThingDisable:        {Symbol: "", Title: "Disable", Description: "Action to disable a thing"},
	ActionThingEnable:         {Symbol: "", Title: "Enable", Description: "Action to enable a thing"},
	ActionThingStop:           {Symbol: "", Title: "Stop", Description: "Stop a running task"},
	ActionMedia:               {Symbol: "", Title: "Media control", Description: "Commands to control media recording and playback"},
	ActionMediaPrevious:       {Symbol: "", Title: "Previous", Description: "Previous track or station"},
	ActionMediaUnmute:         {Symbol: "", Title: "Unmute", Description: "Unmute audio"},
	ActionMediaVolumeIncrease: {Symbol: "", Title: "Increase volume", Description: "Increase volume"},
	ActionSwitch:              {Symbol: "", Title: "Switch", Description: "General switch action"},
	ActionThingStart:          {Symbol: "", Title: "Start", Description: "Start running a task"},
	ActionMediaNext:           {Symbol: "", Title: "Next", Description: "Next track or station"},
	ActionMediaPlay:           {Symbol: "", Title: "Play", Description: "Start or continue playback"},
	ActionDimmerIncrement:     {Symbol: "", Title: "Increase dimmer", Description: ""},
	ActionDimmerSet:           {Symbol: "", Title: "Set dimmer", Description: "Action to set the dimmer value"},
	ActionValveClose:          {Symbol: "", Title: "Close valve", Description: "Action to close the valve"},
	ActionValveOpen:           {Symbol: "", Title: "Open valve", Description: "Action to open the valve"},
	ActionMediaPause:          {Symbol: "", Title: "Pause", Description: "Pause playback"},
	ActionMediaVolumeDecrease: {Symbol: "", Title: "Decrease volume", Description: "Decrease volume"},
	ActionSwitchToggle:        {Symbol: "", Title: "Toggle switch", Description: "Action to toggle the switch"},
	ActionMediaMute:           {Symbol: "", Title: "Mute", Description: "Mute audio"},
	ActionMediaVolume:         {Symbol: "", Title: "Volume", Description: "Set volume level"},
	ActionDimmer:              {Symbol: "", Title: "Dimmer", Description: "General dimmer action"},
	ActionDimmerDecrement:     {Symbol: "", Title: "Lower dimmer", Description: ""},
	ActionSwitchOnOff:         {Symbol: "", Title: "Set On/Off switch", Description: "Action to set the switch on/off state"},
}
