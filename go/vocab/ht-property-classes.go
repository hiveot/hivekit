// Package vocab with HiveOT and WoT vocabulary names for TD Things, properties, events and actions
package vocab

// type: PropertyClasses
// namespace: hiveot
const (
	PropAlarmMotion           = "hiveot:prop:alarm:motion"
	PropAlarmStatus           = "hiveot:prop:alarm:status"
	PropDevice                = "hiveot:prop:device"
	PropDeviceBattery         = "hiveot:prop:device:battery"
	PropDeviceDescription     = "hiveot:prop:device:description"
	PropDeviceEnabledDisabled = "hiveot:prop:device:enabled-disabled"
	PropDeviceFirmwareVersion = "hiveot:prop:device:firmwareversion"
	PropDeviceHardwareVersion = "hiveot:prop:device:hardwareversion"
	PropDeviceMake            = "hiveot:prop:device:make"
	PropDeviceModel           = "hiveot:prop:device:model"
	PropDevicePollinterval    = "hiveot:prop:device:pollinterval"
	PropDeviceSoftwareVersion = "hiveot:prop:device:softwareversion"
	PropDeviceStatus          = "hiveot:prop:device:status"
	PropDeviceTitle           = "hiveot:prop:device:title"
	PropElectric              = "hiveot:prop:electric"
	PropElectricCurrent       = "hiveot:prop:electric:current"
	PropElectricEnergy        = "hiveot:prop:electric:energy"
	PropElectricOverload      = "hiveot:prop:electric:overload"
	PropElectricPower         = "hiveot:prop:electric:poer"
	PropElectricVoltage       = "hiveot:prop:electric:voltage"
	PropEnv                   = "hiveot:prop:env"
	PropEnvAcceleration       = "hiveot:prop:env:acceleration"
	PropEnvAirquality         = "hiveot:prop:env:airquality"
	PropEnvCO                 = "hiveot:prop:env:co"
	PropEnvCO2                = "hiveot:prop:env:co2"
	PropEnvCpuload            = "hiveot:prop:env:cpuload"
	PropEnvDewpoint           = "hiveot:prop:env:dewpoint"
	PropEnvFuelFlowrate       = "hiveot:prop:env:fuel:flowrate"
	PropEnvFuelLevel          = "hiveot:prop:env:fuel:level"
	PropEnvHumidex            = "hiveot:prop:env:humidex"
	PropEnvHumidity           = "hiveot:prop:env:humidity"
	PropEnvLuminance          = "hiveot:prop:env:luminance"
	PropEnvPrecipitation      = "hiveot:prop:env:precipitation"
	PropEnvPrecipitationRain  = "hiveot:prop:env:precipitation:rain"
	PropEnvPrecipitationSnow  = "hiveot:prop:env:precipitation:snow"
	PropEnvPressure           = "hiveot:prop:env:pressure"
	PropEnvPressureSeaLevel   = "hiveot:prop:env:barometer:msl"
	PropEnvPressureSurface    = "hiveot:prop:env:barometer:surface"
	PropEnvTemperature        = "hiveot:prop:env:temperature"
	PropEnvTimezone           = "hiveot:prop:env:timezone"
	PropEnvUV                 = "hiveot:prop:env:uv"
	PropEnvVibration          = "hiveot:prop:env:vibration"
	PropEnvVolume             = "hiveot:prop:env:volume"
	PropEnvWaterFlowrate      = "hiveot:prop:env:water:flowrate"
	PropEnvWaterLevel         = "hiveot:prop:env:water:level"
	PropEnvWindGusts          = "hiveot:prop:env:wind:gusts"
	PropEnvWindHeading        = "hiveot:prop:env:wind:heading"
	PropEnvWindSpeed          = "hiveot:prop:env:wind:speed"
	PropLocation              = "hiveot:prop:location"
	PropLocationCity          = "hiveot:prop:location:city"
	PropLocationLatitude      = "hiveot:prop:location:latitude"
	PropLocationLongitude     = "hiveot:prop:location:longitude"
	PropLocationName          = "hiveot:prop:location:name"
	PropLocationStreet        = "hiveot:prop:location:street"
	PropLocationZipcode       = "hiveot:prop:location:zipcode"
	PropMedia                 = "hiveot:prop:media"
	PropMediaMuted            = "hiveot:prop:media:muted"
	PropMediaPaused           = "hiveot:prop:media:paused"
	PropMediaPlaying          = "hiveot:prop:media:playing"
	PropMediaStation          = "hiveot:prop:media:station"
	PropMediaTrack            = "hiveot:prop:media:track"
	PropMediaVolume           = "hiveot:prop:media:volume"
	PropNet                   = "hiveot:prop:net"
	PropNetAddress            = "hiveot:prop:net:address"
	PropNetConnection         = "hiveot:prop:net:connection"
	PropNetDomainname         = "hiveot:prop:net:domainname"
	PropNetGateway            = "hiveot:prop:net:gateway"
	PropNetHostname           = "hiveot:prop:net:hostname"
	PropNetIP4                = "hiveot:prop:net:ip4"
	PropNetIP6                = "hiveot:prop:net:ip6"
	PropNetLatency            = "hiveot:prop:net:latency"
	PropNetMAC                = "hiveot:prop:net:mac"
	PropNetMask               = "hiveot:prop:net:mask"
	PropNetPort               = "hiveot:prop:net:port"
	PropNetSignalstrength     = "hiveot:prop:net:signalstrength"
	PropNetSubnet             = "hiveot:prop:net:subnet"
	PropStatusOnOff           = "hiveot:prop:status:onoff"
	PropStatusOpenClosed      = "hiveot:prop:status:openclosed"
	PropStatusStartedStopped  = "hiveot:prop:status:started-stopped"
	PropStatusYesNo           = "hiveot:prop:status:yes-no"
	PropSwitch                = "hiveot:prop:switch"
	PropSwitchDimmer          = "hiveot:prop:switch:dimmer"
	PropSwitchLight           = "hiveot:prop:switch:light"
	PropSwitchLocked          = "hiveot:prop:switch:locked"
	PropSwitchOnOff           = "hiveot:prop:switch:onoff"
)

// end of PropertyClasses

// PropertyClassesMap maps @type to symbol, title and description
var PropertyClassesMap = map[string]struct {
	Symbol      string
	Title       string
	Description string
}{
	PropNetLatency:            {Symbol: "", Title: "Network latency", Description: "Delay between hub and client"},
	PropSwitch:                {Symbol: "", Title: "Switch status", Description: ""},
	PropAlarmMotion:           {Symbol: "", Title: "Motion", Description: "Motion detected"},
	PropDeviceBattery:         {Symbol: "", Title: "Battery level", Description: "Device battery level"},
	PropDeviceEnabledDisabled: {Symbol: "", Title: "Enabled/Disabled", Description: "Enabled or disabled state"},
	PropDeviceHardwareVersion: {Symbol: "", Title: "Hardware version", Description: ""},
	PropEnv:                   {Symbol: "", Title: "Environmental property", Description: "Property of environmental sensor"},
	PropMedia:                 {Symbol: "", Title: "Media commands", Description: "Control of media equipment"},
	PropMediaVolume:           {Symbol: "", Title: "Volume", Description: "Media volume setting"},
	PropNetAddress:            {Symbol: "", Title: "Address", Description: "Network address"},
	PropEnvFuelFlowrate:       {Symbol: "", Title: "Fuel flow rate", Description: ""},
	PropEnvLuminance:          {Symbol: "", Title: "Luminance", Description: ""},
	PropEnvUV:                 {Symbol: "", Title: "UV", Description: ""},
	PropNetMask:               {Symbol: "", Title: "Netmask", Description: "Network mask. Example: 255.255.255.0 or 24/8"},
	PropSwitchDimmer:          {Symbol: "", Title: "Dimmer value", Description: ""},
	PropEnvWaterFlowrate:      {Symbol: "", Title: "Water flow rate", Description: ""},
	PropEnvWindGusts:          {Symbol: "", Title: "Wind gusts", Description: "Speed of wind gusts"},
	PropDeviceStatus:          {Symbol: "", Title: "Status", Description: "Device status; alive, awake, dead, sleeping"},
	PropDevice:                {Symbol: "", Title: "Device attributes", Description: "Attributes describing a device"},
	PropDeviceSoftwareVersion: {Symbol: "", Title: "Software version", Description: ""},
	PropEnvCO2:                {Symbol: "", Title: "Carbon dioxide level", Description: "Carbon dioxide level"},
	PropNetPort:               {Symbol: "", Title: "Port", Description: "Network port"},
	PropElectric:              {Symbol: "", Title: "Electrical properties", Description: "General group of electrical properties"},
	PropLocationCity:          {Symbol: "", Title: "City", Description: "City name"},
	PropNetMAC:                {Symbol: "", Title: "MAC", Description: "Hardware MAC address"},
	PropNetSignalstrength:     {Symbol: "", Title: "Signal strength", Description: "Wireless signal strength"},
	PropNetDomainname:         {Symbol: "", Title: "Domain name", Description: "Domainname of the client"},
	PropNetIP4:                {Symbol: "", Title: "IP4 address", Description: "Device IP4 address"},
	PropStatusStartedStopped:  {Symbol: "", Title: "Started/Stopped", Description: "Started or stopped status"},
	PropEnvCO:                 {Symbol: "", Title: "Carbon monoxide level", Description: "Carbon monoxide level"},
	PropEnvPressure:           {Symbol: "", Title: "Pressure", Description: ""},
	PropDeviceMake:            {Symbol: "", Title: "Make", Description: "Device manufacturer"},
	PropEnvDewpoint:           {Symbol: "", Title: "Dew point", Description: "Dew point temperature"},
	PropEnvVibration:          {Symbol: "", Title: "Vibration", Description: ""},
	PropDeviceTitle:           {Symbol: "", Title: "Title", Description: "Device friendly title"},
	PropEnvPressureSurface:    {Symbol: "", Title: "Surface level pressure", Description: "Surface level atmospheric pressure"},
	PropLocationStreet:        {Symbol: "", Title: "Street", Description: "Street address"},
	PropNet:                   {Symbol: "", Title: "Network properties", Description: "General network properties"},
	PropStatusOnOff:           {Symbol: "", Title: "On/off status", Description: ""},
	PropSwitchOnOff:           {Symbol: "", Title: "On/Off switch", Description: ""},
	PropEnvPrecipitationSnow:  {Symbol: "", Title: "Snow precipitation", Description: "Precipitation as snow"},
	PropEnvPressureSeaLevel:   {Symbol: "", Title: "Sea level pressure", Description: "Sea level equivalent atmospheric pressure"},
	PropMediaTrack:            {Symbol: "", Title: "Track", Description: "Selected A/V track"},
	PropDeviceFirmwareVersion: {Symbol: "", Title: "Firmware version", Description: ""},
	PropElectricEnergy:        {Symbol: "", Title: "Energy", Description: "Electrical energy consumed"},
	PropElectricOverload:      {Symbol: "", Title: "Overload protection", Description: "Cut load on overload"},
	PropEnvCpuload:            {Symbol: "", Title: "CPU load level", Description: "Device CPU load level"},
	PropLocationLongitude:     {Symbol: "", Title: "Longitude", Description: "Longitude geographic coordinate"},
	PropMediaStation:          {Symbol: "", Title: "Station", Description: "Selected radio station"},
	PropLocationName:          {Symbol: "", Title: "Location name", Description: "Name of the location"},
	PropElectricCurrent:       {Symbol: "", Title: "Current", Description: "Electrical current"},
	PropEnvHumidex:            {Symbol: "", Title: "Humidex", Description: ""},
	PropEnvPrecipitation:      {Symbol: "", Title: "Precipitation", Description: "Total precipitation of rain and snow"},
	PropEnvTimezone:           {Symbol: "", Title: "Timezone", Description: ""},
	PropMediaPlaying:          {Symbol: "", Title: "Playing", Description: "Media is playing"},
	PropMediaMuted:            {Symbol: "", Title: "Muted", Description: "Audio is muted"},
	PropNetGateway:            {Symbol: "", Title: "Gateway", Description: "Network gateway address"},
	PropElectricPower:         {Symbol: "", Title: "Power", Description: "Electrical power being consumed"},
	PropEnvAcceleration:       {Symbol: "", Title: "Acceleration", Description: ""},
	PropEnvPrecipitationRain:  {Symbol: "", Title: "Rain precipitation", Description: "Precipitation as rain"},
	PropEnvTemperature:        {Symbol: "", Title: "Temperature", Description: ""},
	PropEnvWaterLevel:         {Symbol: "", Title: "Water level", Description: ""},
	PropLocationZipcode:       {Symbol: "", Title: "Zip code", Description: "Location ZIP code"},
	PropNetHostname:           {Symbol: "", Title: "Hostname", Description: "Hostname of the client"},
	PropNetIP6:                {Symbol: "", Title: "IP6 address", Description: "Device IP6 address"},
	PropDevicePollinterval:    {Symbol: "", Title: "Polling interval", Description: "Interval to poll for updates"},
	PropElectricVoltage:       {Symbol: "", Title: "Voltage", Description: "Electrical voltage potential"},
	PropEnvAirquality:         {Symbol: "", Title: "Air quality", Description: "Air quality level"},
	PropNetConnection:         {Symbol: "", Title: "Connection", Description: "Connection status, connected, connecting, retrying, disconnected,..."},
	PropStatusYesNo:           {Symbol: "", Title: "Yes/No", Description: "Status with yes or no value"},
	PropSwitchLocked:          {Symbol: "", Title: "Lock", Description: "Electric lock status"},
	PropDeviceDescription:     {Symbol: "", Title: "Description", Description: "Device product description"},
	PropEnvVolume:             {Symbol: "", Title: "Volume", Description: ""},
	PropEnvWindHeading:        {Symbol: "", Title: "Wind heading", Description: "Direction wind is heading"},
	PropEnvWindSpeed:          {Symbol: "", Title: "Wind speed", Description: "Average speed of wind"},
	PropLocationLatitude:      {Symbol: "", Title: "Latitude", Description: "Latitude geographic coordinate"},
	PropNetSubnet:             {Symbol: "", Title: "Subnet", Description: "Network subnet address. Example: 192.168.0.0"},
	PropSwitchLight:           {Symbol: "", Title: "Light switch", Description: ""},
	PropLocation:              {Symbol: "", Title: "Location", Description: "General location information"},
	PropMediaPaused:           {Symbol: "", Title: "Paused", Description: "Media is paused"},
	PropStatusOpenClosed:      {Symbol: "", Title: "Open/Closed status", Description: ""},
	PropAlarmStatus:           {Symbol: "", Title: "Alarm state", Description: "Current alarm status"},
	PropDeviceModel:           {Symbol: "", Title: "Model", Description: "Device model"},
	PropEnvFuelLevel:          {Symbol: "", Title: "Fuel level", Description: ""},
	PropEnvHumidity:           {Symbol: "", Title: "Humidity", Description: ""},
}
