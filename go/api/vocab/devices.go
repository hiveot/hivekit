package vocab

// Device types for use in TD @type fields.
// Intended to identify basic capabilities of devices.
// Future: link each @type to a TM that specify these capabilities.
// namespace: hiveot
const (
	Device                         = "hiveot:device"
	DeviceActuator                 = "hiveot:actuator"
	DeviceActuatorAlarm            = "hiveot:actuator:alarm"
	DeviceActuatorLight            = "hiveot:actuator:light"
	DeviceActuatorLock             = "hiveot:actuator:lock"
	DeviceActuatorMotor            = "hiveot:actuator:motor"
	DeviceActuatorOutput           = "hiveot:actuator:output"
	DeviceActuatorRanged           = "hiveot:actuator:ranged"
	DeviceActuatorRelay            = "hiveot:actuator:relay"
	DeviceActuatorValve            = "hiveot:actuator:valve"
	DeviceActuatorValveFuel        = "hiveot:actuator:valve:fuel"
	DeviceActuatorValveWater       = "hiveot:actuator:valve:water"
	DeviceAppliance                = "hiveot:appliance"
	DeviceApplianceDishwasher      = "hiveot:dishwasher"
	DeviceApplianceDryer           = "hiveot:dryer"
	DeviceApplianceFreezer         = "hiveot:freezer"
	DeviceApplianceFridge          = "hiveot:fridge"
	DeviceApplianceWasher          = "hiveot:washer"
	DeviceBatteryMonitor           = "hiveot:monitor"
	DeviceComputer                 = "hiveot:computer"
	DeviceComputerCellphone        = "hiveot:cellphone"
	DeviceComputerEmbedded         = "hiveot:computer:embedded"
	DeviceComputerMemory           = "hiveot:computer:memory"
	DeviceComputerPC               = "hiveot:pc"
	DeviceComputerSatPhone         = "hiveot:satphone"
	DeviceComputerTablet           = "hiveot:tablet"
	DeviceComputerVoipPhone        = "hiveot:voipphone"
	DeviceControlClimate           = "hiveot:control:climate"
	DeviceControlDimmer            = "hiveot:control:dimmer"
	DeviceControlIrrigation        = "hiveot:control:irrigation"
	DeviceControlJoystick          = "hiveot:control:joystick"
	DeviceControlKeypad            = "hiveot:control:keypad"
	DeviceControlPool              = "hiveot:control:pool"
	DeviceControlPushbutton        = "hiveot:control:pushbutton"
	DeviceControlSwitch            = "hiveot:control:switch"
	DeviceControlThermostat        = "hiveot:control:thermostat"
	DeviceControlToggle            = "hiveot:control:toggle"
	DeviceGateway                  = "hiveot:gateway"
	DeviceGatewayCoap              = "hiveot:gateway:coap"
	DeviceGatewayInsteon           = "hiveot:gateway:insteon"
	DeviceGatewayLora              = "hiveot:gateway:lora"
	DeviceGatewayOnewire           = "hiveot:gateway:onewire"
	DeviceGatewayZigbee            = "hiveot:gateway:zigbee"
	DeviceGatewayZwave             = "hiveot:gateway:zwave"
	DeviceIndicator                = "hiveot:indicator"
	DeviceMedia                    = "hiveot:media"
	DeviceMediaAmplifier           = "hiveot:media:amplifier"
	DeviceMediaCamera              = "hiveot:media:camera"
	DeviceMediaMicrophone          = "hiveot:media:microphone"
	DeviceMediaPlayer              = "hiveot:media:player"
	DeviceMediaRadio               = "hiveot:media:radio"
	DeviceMediaReceiver            = "hiveot:media:receiver"
	DeviceMediaSpeaker             = "hiveot:media:speaker"
	DeviceMediaTV                  = "hiveot:media:tv"
	DeviceMeter                    = "hiveot:meter"
	DeviceMeterElectric            = "hiveot:meter:electric"
	DeviceMeterElectricCurrent     = "hiveot:meter:electric:current"
	DeviceMeterElectricEnergy      = "hiveot:meter:electric:energy"
	DeviceMeterElectricPower       = "hiveot:meter:electric:power"
	DeviceMeterElectricVoltage     = "hiveot:meter:electric:voltage"
	DeviceMeterFuel                = "hiveot:meter:fuel"
	DeviceMeterFuelFlow            = "hiveot:meter:fuel:flow"
	DeviceMeterFuelLevel           = "hiveot:meter:fuel:level"
	DeviceMeterWater               = "hiveot:meter:water"
	DeviceMeterWaterConsumption    = "hiveot:meter:water:consumption"
	DeviceMeterWaterFlow           = "hiveot:meter:water:flow"
	DeviceMeterWaterLevel          = "hiveot:meter:water:level"
	DeviceMeterWind                = "hiveot:meter:wind"
	DeviceNet                      = "hiveot:net"
	DeviceNetBluetooth             = "hiveot:net:bluetooth"
	DeviceNetGateway               = "hiveot:net:gateway"
	DeviceNetRouter                = "hiveot:net:router"
	DeviceNetSwitch                = "hiveot:net:switch"
	DeviceNetWifi                  = "hiveot:net:wifi"
	DeviceNetWifiAp                = "hiveot:net:wifi:ap"
	DeviceSensor                   = "hiveot:sensor"
	DeviceSensorEnvironment        = "hiveot:sensor:environment"
	DeviceSensorInput              = "hiveot:sensor:input"
	DeviceSensorMulti              = "hiveot:sensor:multi"
	DeviceSensorScale              = "hiveot:sensor:scale"
	DeviceSensorSecurity           = "hiveot:sensor:security"
	DeviceSensorSecurityDoorWindow = "hiveot:sensor:security:doorwindow"
	DeviceSensorSecurityGlass      = "hiveot:sensor:security:glass"
	DeviceSensorSecurityMotion     = "hiveot:sensor:security:motion"
	DeviceSensorSmoke              = "hiveot:sensor:smoke"
	DeviceSensorSound              = "hiveot:sensor:sound"
	DeviceSensorThermometer        = "hiveot:sensor:thermometer"
	DeviceSensorWaterLeak          = "hiveot:sensor:water:leak"
	DeviceTime                     = "hiveot:time"
	DeviceTypeService              = "hiveot:service"
)

// end of DeviceClasses

// DeviceClassesMap maps @type to symbol, title and description
var DeviceClassesMap = map[string]struct {
	Symbol      string
	Title       string
	Description string
}{
	Device: {Symbol: "", Title: "Device", Description: "Generic device"},

	DeviceActuator:           {Symbol: "", Title: "Actuator", Description: "Generic actuator"},
	DeviceActuatorAlarm:      {Symbol: "", Title: "Alarm", Description: "Siren or light alarm"},
	DeviceActuatorLight:      {Symbol: "", Title: "Light", Description: "Smart LED or other light"},
	DeviceActuatorLock:       {Symbol: "", Title: "Lock", Description: "Electronic door lock"},
	DeviceActuatorMotor:      {Symbol: "", Title: "Motor", Description: "Motor driven actuator, such as garage door, blinds, tv lifts"},
	DeviceActuatorOutput:     {Symbol: "", Title: "Output", Description: "General purpose electrical output signal"},
	DeviceActuatorRanged:     {Symbol: "", Title: "Ranged actuator", Description: "Generic ranged actuator with a set point"},
	DeviceActuatorRelay:      {Symbol: "", Title: "Relay", Description: "Generic relay electrical switch"},
	DeviceActuatorValve:      {Symbol: "", Title: "Valve", Description: "Electric powered valve for fluids or gas"},
	DeviceActuatorValveFuel:  {Symbol: "", Title: "Fuel valve", Description: "Electric powered fuel valve"},
	DeviceActuatorValveWater: {Symbol: "", Title: "Water valve", Description: "Electric powered water valve"},

	DeviceAppliance:           {Symbol: "", Title: "Appliance", Description: "Appliance to accomplish a particular task for occupant use"},
	DeviceApplianceDishwasher: {Symbol: "", Title: "Dishwasher", Description: "Dishwasher"},
	DeviceApplianceDryer:      {Symbol: "", Title: "Dryer", Description: "CloDevice dryer"},
	DeviceApplianceFreezer:    {Symbol: "", Title: "Freezer", Description: "Refrigerator freezer"},
	DeviceApplianceFridge:     {Symbol: "", Title: "Fridge", Description: "Refrigerator appliance"},
	DeviceApplianceWasher:     {Symbol: "", Title: "Washer", Description: "CloDevice washer"},

	DeviceBatteryMonitor: {Symbol: "", Title: "Battery Monitor", Description: "Battery monitor and charge controller"},

	DeviceComputer:          {Symbol: "", Title: "Computing Device", Description: "General purpose computing device"},
	DeviceComputerCellphone: {Symbol: "", Title: "Cell Phone", Description: "Cellular phone"},
	DeviceComputerEmbedded:  {Symbol: "", Title: "Embedded System", Description: "Embedded computing device"},
	DeviceComputerMemory:    {Symbol: "", Title: "Memory", Description: "Stand-alone memory device such as eeprom or iButtons"},
	DeviceComputerPC:        {Symbol: "", Title: "PC/Laptop", Description: "Personal computer/laptop"},
	DeviceComputerSatPhone:  {Symbol: "", Title: "Satellite phone", Description: ""},
	DeviceComputerTablet:    {Symbol: "", Title: "Tablet", Description: "Tablet computer"},
	DeviceComputerVoipPhone: {Symbol: "", Title: "VoIP Phone", Description: "Voice over IP phone"},

	DeviceControlClimate:    {Symbol: "", Title: "Climate control", Description: "Device for controlling climate of a space"},
	DeviceControlDimmer:     {Symbol: "", Title: "Dimmer control", Description: "Light dimmer input device"},
	DeviceControlIrrigation: {Symbol: "", Title: "Irrigation control", Description: "Device for control of an irrigation system"},
	DeviceControlJoystick:   {Symbol: "", Title: "Joystick", Description: "Flight control stick"},
	DeviceControlKeypad:     {Symbol: "", Title: "Keypad", Description: "Multi-key pad for command input"},
	DeviceControlPool:       {Symbol: "", Title: "Pool control", Description: "Device for controlling pool settings"},
	DeviceControlPushbutton: {Symbol: "", Title: "Momentary switch", Description: "Momentary push button control input"},
	DeviceControlSwitch:     {Symbol: "", Title: "Input switch", Description: "On or off switch input control"},
	DeviceControlThermostat: {Symbol: "", Title: "Thermostat", Description: "Thermostat HVAC control"},
	DeviceControlToggle:     {Symbol: "", Title: "Toggle switch", Description: "Toggle switch input control"},

	DeviceIndicator: {Symbol: "", Title: "Indicator", Description: "Visual or audio indicator device"},

	DeviceMedia:           {Symbol: "", Title: "A/V media", Description: "Generic device for audio/video media record or playback"},
	DeviceMediaAmplifier:  {Symbol: "", Title: "Audio amplifier", Description: "Audio amplifier with volume controls"},
	DeviceMediaCamera:     {Symbol: "", Title: "Camera", Description: "Video camera"},
	DeviceMediaMicrophone: {Symbol: "", Title: "Microphone", Description: "Microphone for capturing audio"},
	DeviceMediaPlayer:     {Symbol: "", Title: "Media player", Description: "CD/DVD/Blueray/USB player of recorded media"},
	DeviceMediaRadio:      {Symbol: "", Title: "Radio", Description: "AM or FM radio receiver"},
	DeviceMediaReceiver:   {Symbol: "", Title: "Receiver", Description: "Audio/video receiver and player"},
	DeviceMediaSpeaker:    {Symbol: "", Title: "Connected speakers", Description: "Network connected speakers"},
	DeviceMediaTV:         {Symbol: "", Title: "TV", Description: "Network connected television"},

	DeviceMeter:                 {Symbol: "", Title: "Meter", Description: "General metering device"},
	DeviceMeterElectric:         {Symbol: "", Title: "", Description: ""},
	DeviceMeterElectricCurrent:  {Symbol: "", Title: "Electric current", Description: "Electrical current meter"},
	DeviceMeterElectricEnergy:   {Symbol: "", Title: "Electric energy", Description: "Electrical energy meter"},
	DeviceMeterElectricPower:    {Symbol: "", Title: "Electrical Power", Description: "Electrical power meter"},
	DeviceMeterElectricVoltage:  {Symbol: "", Title: "Voltage", Description: "Electrical voltage meter"},
	DeviceMeterFuel:             {Symbol: "", Title: "Fuel metering device", Description: "General fuel metering device"},
	DeviceMeterFuelFlow:         {Symbol: "", Title: "Fuel flow rate", Description: "Dedicated fuel flow rate metering device"},
	DeviceMeterFuelLevel:        {Symbol: "", Title: "Fuel level", Description: "Dedicated fuel level metering device"},
	DeviceMeterWater:            {Symbol: "", Title: "Water metering device", Description: "General water metering device"},
	DeviceMeterWind:             {Symbol: "", Title: "Wind", Description: "Dedicated wind meter"},
	DeviceMeterWaterConsumption: {Symbol: "", Title: "Water consumption meter", Description: "Water consumption meter"},
	DeviceMeterWaterFlow:        {Symbol: "", Title: "Water flow", Description: "Dedicated water flow-rate meter"},
	DeviceMeterWaterLevel:       {Symbol: "", Title: "Water level", Description: "Dedicated water level meter"},

	DeviceNet:            {Symbol: "", Title: "Network device", Description: "Generic network device"},
	DeviceNetBluetooth:   {Symbol: "", Title: "Bluetooth", Description: "Bluetooth radio"},
	DeviceGateway:        {Symbol: "", Title: "Gateway", Description: "Generic gateway device providing access to other devices"},
	DeviceGatewayCoap:    {Symbol: "", Title: "CoAP gateway", Description: "Gateway providing access to CoAP devices"},
	DeviceGatewayInsteon: {Symbol: "", Title: "Insteon gateway", Description: "Gateway providing access to Insteon devices"},
	DeviceGatewayLora:    {Symbol: "", Title: "LoRaWAN gateway", Description: "Gateway providing access to LoRa devices"},
	DeviceGatewayOnewire: {Symbol: "", Title: "1-Wire gateway", Description: "Gateway providing access to 1-wire devices"},
	DeviceGatewayZigbee:  {Symbol: "", Title: "Zigbee gateway", Description: "Gateway providing access to Zigbee devices"},
	DeviceGatewayZwave:   {Symbol: "", Title: "ZWave gateway", Description: "Gateway providing access to ZWave devices"},
	DeviceNetRouter:      {Symbol: "", Title: "Network router", Description: "IP Network router providing access to other IP networks"},
	DeviceNetSwitch:      {Symbol: "", Title: "Network switch", Description: "Network switch to connect computer devices to the network"},
	DeviceNetWifi:        {Symbol: "", Title: "Wifi device", Description: "Generic wifi device"},
	DeviceNetWifiAp:      {Symbol: "", Title: "Wifi access point", Description: "Wireless access point for IP networks"},

	DeviceSensor:                   {Symbol: "", Title: "Sensor", Description: "Generic sensor device"},
	DeviceSensorEnvironment:        {Symbol: "", Title: "Environmental sensor", Description: "Environmental sensor with one or more features such as temperature, humidity, etc"},
	DeviceSensorInput:              {Symbol: "", Title: "Input sensor", Description: "General purpose electrical input sensor"},
	DeviceSensorMulti:              {Symbol: "", Title: "Multi sensor", Description: "Sense multiple inputs"},
	DeviceSensorScale:              {Symbol: "", Title: "Scale", Description: "Electronic weigh scale"},
	DeviceSensorSecurity:           {Symbol: "", Title: "Security", Description: "Generic security sensor"},
	DeviceSensorSecurityDoorWindow: {Symbol: "", Title: "Door/Window sensor", Description: "Dedicated door/window opening security sensor"},
	DeviceSensorSecurityGlass:      {Symbol: "", Title: "Glass sensor", Description: "Dedicated sensor for detecting breaking of glass"},
	DeviceSensorSecurityMotion:     {Symbol: "", Title: "Motion sensor", Description: "Dedicated security sensor detecting motion"},
	DeviceSensorSound:              {Symbol: "", Title: "Sound detector", Description: ""},
	DeviceSensorSmoke:              {Symbol: "", Title: "Smoke detector", Description: ""},
	DeviceSensorThermometer:        {Symbol: "", Title: "Thermometer", Description: "Environmental thermometer"},
	DeviceSensorWaterLeak:          {Symbol: "", Title: "Water leak detector", Description: "Dedicated water leak detector"},

	DeviceTime:        {Symbol: "", Title: "Clock", Description: "Time tracking device such as clocks and time chips"},
	DeviceTypeService: {Symbol: "", Title: "Service", Description: "General service for processing data and offering features of interest"},
}
