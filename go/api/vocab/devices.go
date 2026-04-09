package vocab

// type: Device types
// namespace: hiveot
const (
	Device                         = "hiveot:device"
	DeviceActuator                 = "hiveot:device:actuator"
	DeviceActuatorAlarm            = "hiveot:device:actuator:alarm"
	DeviceActuatorBeacon           = "hiveot:device:actuator:beacon"
	DeviceActuatorDimmer           = "hiveot:device:actuator:dimmer"
	DeviceActuatorLight            = "hiveot:device:actuator:light"
	DeviceActuatorLock             = "hiveot:device:actuator:lock"
	DeviceActuatorMotor            = "hiveot:device:actuator:motor"
	DeviceActuatorOutput           = "hiveot:device:actuator:output"
	DeviceActuatorRanged           = "hiveot:device:actuator:ranged"
	DeviceActuatorRelay            = "hiveot:device:actuator:relay"
	DeviceActuatorSwitch           = "hiveot:device:actuator:switch"
	DeviceActuatorValve            = "hiveot:device:actuator:valve"
	DeviceActuatorValveFuel        = "hiveot:device:actuator:valve:fuel"
	DeviceActuatorValveWater       = "hiveot:device:actuator:valve:water"
	DeviceAppliance                = "hiveot:device:appliance"
	DeviceApplianceDishwasher      = "hiveot:device:appliance:dishwasher"
	DeviceApplianceDryer           = "hiveot:device:appliance:dryer"
	DeviceApplianceFreezer         = "hiveot:device:appliance:freezer"
	DeviceApplianceFridge          = "hiveot:device:appliance:fridge"
	DeviceApplianceWasher          = "hiveot:device:appliance:washer"
	DeviceBatteryMonitor           = "hiveot:device:battery:monitor"
	DeviceComputer                 = "hiveot:device:computer"
	DeviceComputerCellphone        = "hiveot:device:computer:cellphone"
	DeviceComputerEmbedded         = "hiveot:device:computer:embedded"
	DeviceComputerMemory           = "hiveot:device:computer:memory"
	DeviceComputerPC               = "hiveot:device:computer:pc"
	DeviceComputerPotsPhone        = "hiveot:device:computer:potsphone"
	DeviceComputerSatPhone         = "hiveot:device:computer:satphone"
	DeviceComputerTablet           = "hiveot:device:computer:tablet"
	DeviceComputerVoipPhone        = "hiveot:device:computer:voipphone"
	DeviceControl                  = "hiveot:device:control"
	DeviceControlClimate           = "hiveot:device:control:climate"
	DeviceControlDimmer            = "hiveot:device:control:dimmer"
	DeviceControlIrrigation        = "hiveot:device:control:irrigation"
	DeviceControlJoystick          = "hiveot:device:control:joystick"
	DeviceControlKeypad            = "hiveot:device:control:keypad"
	DeviceControlPool              = "hiveot:device:control:pool"
	DeviceControlPushbutton        = "hiveot:device:control:pushbutton"
	DeviceControlSwitch            = "hiveot:device:control:switch"
	DeviceControlThermostat        = "hiveot:device:control:thermostat"
	DeviceControlToggle            = "hiveot:device:control:toggle"
	DeviceIndicator                = "hiveot:device:indicator"
	DeviceMedia                    = "hiveot:device:media"
	DeviceMediaAmplifier           = "hiveot:device:media:amplifier"
	DeviceMediaCamera              = "hiveot:device:media:camera"
	DeviceMediaMicrophone          = "hiveot:device:media:microphone"
	DeviceMediaPlayer              = "hiveot:device:media:player"
	DeviceMediaRadio               = "hiveot:device:media:radio"
	DeviceMediaReceiver            = "hiveot:device:media:receiver"
	DeviceMediaSpeaker             = "hiveot:device:media:speaker"
	DeviceMediaTV                  = "hiveot:device:media:tv"
	DeviceMeter                    = "hiveot:device:meter"
	DeviceMeterElectric            = "hiveot:device:meter:electric"
	DeviceMeterElectricCurrent     = "hiveot:device:meter:electric:current"
	DeviceMeterElectricEnergy      = "hiveot:device:meter:electric:energy"
	DeviceMeterElectricPower       = "hiveot:device:meter:electric:power"
	DeviceMeterElectricVoltage     = "hiveot:device:meter:electric:voltage"
	DeviceMeterFuel                = "hiveot:device:meter:fuel"
	DeviceMeterFuelFlow            = "hiveot:device:meter:fuel:flow"
	DeviceMeterFuelLevel           = "hiveot:device:meter:fuel:level"
	DeviceMeterWater               = "hiveot:device:meter:water"
	DeviceMeterWaterConsumption    = "hiveot:device:meter:water:consumption"
	DeviceMeterWaterFlow           = "hiveot:device:meter:water:flow"
	DeviceMeterWaterLevel          = "hiveot:device:meter:water:level"
	DeviceMeterWind                = "hiveot:device:meter:wind"
	DeviceNet                      = "hiveot:device:net"
	DeviceNetBluetooth             = "hiveot:device:net:bluetooth"
	DeviceNetGateway               = "hiveot:device:net:gateway"
	DeviceNetGatewayCoap           = "hiveot:device:net:gateway:coap"
	DeviceNetGatewayInsteon        = "hiveot:device:net:gateway:insteon"
	DeviceNetGatewayLora           = "hiveot:device:net:gateway:lora"
	DeviceNetGatewayOnewire        = "hiveot:device:net:gateway:onewire"
	DeviceNetGatewayZigbee         = "hiveot:device:net:gateway:zigbee"
	DeviceNetGatewayZwave          = "hiveot:device:net:gateway:zwave"
	DeviceNetLora                  = "hiveot:device:net:lora"
	DeviceNetLoraP2P               = "hiveot:device:net:lora:p2p"
	DeviceNetRouter                = "hiveot:device:net:router"
	DeviceNetSwitch                = "hiveot:device:net:switch"
	DeviceNetWifi                  = "hiveot:device:net:wifi"
	DeviceNetWifiAp                = "hiveot:device:net:wifi:ap"
	DeviceSensor                   = "hiveot:device:sensor"
	DeviceSensorEnvironment        = "hiveot:device:sensor:environment"
	DeviceSensorInput              = "hiveot:device:sensor:input"
	DeviceSensorMulti              = "hiveot:device:sensor:multi"
	DeviceSensorScale              = "hiveot:device:sensor:scale"
	DeviceSensorSecurity           = "hiveot:device:sensor:security"
	DeviceSensorSecurityDoorWindow = "hiveot:device:sensor:security:doorwindow"
	DeviceSensorSecurityGlass      = "hiveot:device:sensor:security:glass"
	DeviceSensorSecurityMotion     = "hiveot:device:sensor:security:motion"
	DeviceSensorSmoke              = "hiveot:device:sensor:smoke"
	DeviceSensorSound              = "hiveot:device:sensor:sound"
	DeviceSensorThermometer        = "hiveot:device:sensor:thermometer"
	DeviceSensorWaterLeak          = "hiveot:device:sensor:water:leak"
	DeviceTime                     = "hiveot:device:time"
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
	DeviceActuatorBeacon:     {Symbol: "", Title: "Beacon", Description: "Location beacon"},
	DeviceActuatorDimmer:     {Symbol: "", Title: "Dimmer", Description: "Light dimmer"},
	DeviceActuatorLight:      {Symbol: "", Title: "Light", Description: "Smart LED or other light"},
	DeviceActuatorLock:       {Symbol: "", Title: "Lock", Description: "Electronic door lock"},
	DeviceActuatorMotor:      {Symbol: "", Title: "Motor", Description: "Motor driven actuator, such as garage door, blinds, tv lifts"},
	DeviceActuatorOutput:     {Symbol: "", Title: "Output", Description: "General purpose electrical output signal"},
	DeviceActuatorRanged:     {Symbol: "", Title: "Ranged actuator", Description: "Generic ranged actuator with a set point"},
	DeviceActuatorRelay:      {Symbol: "", Title: "Relay", Description: "Generic relay electrical switch"},
	DeviceActuatorSwitch:     {Symbol: "", Title: "Switch", Description: "An electric powered on/off switch for powering circuits"},
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
	DeviceComputerPotsPhone: {Symbol: "", Title: "Land Line", Description: "Plain Old Telephone System, aka landline"},
	DeviceComputerSatPhone:  {Symbol: "", Title: "Satellite phone", Description: ""},
	DeviceComputerTablet:    {Symbol: "", Title: "Tablet", Description: "Tablet computer"},
	DeviceComputerVoipPhone: {Symbol: "", Title: "VoIP Phone", Description: "Voice over IP phone"},

	DeviceControl:           {Symbol: "", Title: "Input controller", Description: "Generic input controller"},
	DeviceControlClimate:    {Symbol: "", Title: "Climate control", Description: "Device for controlling climate of a space"},
	DeviceControlDimmer:     {Symbol: "", Title: "Dimmer", Description: "Light dimmer input device"},
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

	DeviceNet:               {Symbol: "", Title: "Network device", Description: "Generic network device"},
	DeviceNetBluetooth:      {Symbol: "", Title: "Bluetooth", Description: "Bluetooth radio"},
	DeviceNetGateway:        {Symbol: "", Title: "Gateway", Description: "Generic gateway device providing access to other devices"},
	DeviceNetGatewayCoap:    {Symbol: "", Title: "CoAP gateway", Description: "Gateway providing access to CoAP devices"},
	DeviceNetGatewayInsteon: {Symbol: "", Title: "Insteon gateway", Description: "Gateway providing access to Insteon devices"},
	DeviceNetGatewayLora:    {Symbol: "", Title: "LoRaWAN gateway", Description: "Gateway providing access to LoRa devices"},
	DeviceNetGatewayOnewire: {Symbol: "", Title: "1-Wire gateway", Description: "Gateway providing access to 1-wire devices"},
	DeviceNetGatewayZigbee:  {Symbol: "", Title: "Zigbee gateway", Description: "Gateway providing access to Zigbee devices"},
	DeviceNetGatewayZwave:   {Symbol: "", Title: "ZWave gateway", Description: "Gateway providing access to ZWave devices"},
	DeviceNetLora:           {Symbol: "", Title: "LoRa network device", Description: "Generic Long Range network protocol device"},
	DeviceNetLoraP2P:        {Symbol: "", Title: "LoRa P2P", Description: "LoRa Peer-to-peer network device"},
	DeviceNetRouter:         {Symbol: "", Title: "Network router", Description: "IP DeviceNetwork router providing access to other IP networks"},
	DeviceNetSwitch:         {Symbol: "", Title: "Network switch", Description: "Network switch to connect computer devices to the network"},
	DeviceNetWifi:           {Symbol: "", Title: "Wifi device", Description: "Generic wifi device"},
	DeviceNetWifiAp:         {Symbol: "", Title: "Wifi access point", Description: "Wireless access point for IP networks"},

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
