package vocab

// type: UnitClasses
// namespace: hiveot
const (
	UnitAmpere           = "hiveot:unit:ampere"
	UnitCandela          = "hiveot:unit:candela"
	UnitCelcius          = "hiveot:unit:celcius"
	UnitCount            = "hiveot:unit:count"
	UnitDegree           = "hiveot:unit:degree"
	UnitFahrenheit       = "hiveot:unit:fahrenheit"
	UnitFoot             = "hiveot:unit:foot"
	UnitGallon           = "hiveot:unit:gallon"
	UnitHectoPascal      = "hiveot:unit:hectopascal"
	UnitKelvin           = "hiveot:unit:kelvin"
	UnitKilogram         = "hiveot:unit:kilogram"
	UnitKilometerPerHour = "hiveot:unit:kph"
	UnitKilowattHour     = "hiveot:unit:kilowatthour"
	UnitLiter            = "hiveot:unit:liter"
	UnitLumen            = "hiveot:unit:lumen"
	UnitLux              = "hiveot:unit:lux"
	UnitMercury          = "hiveot:unit:mercury"
	UnitMeter            = "hiveot:unit:meter"
	UnitMeterPerSecond   = "hiveot:unit:meterspersecond"
	UnitMilesPerHour     = "hiveot:unit:milesperhour"
	UnitMilliBar         = "hiveot:unit:millibar"
	UnitMilliMeter       = "hiveot:unit:millimeter"
	UnitMilliSecond      = "hiveot:unit:millisecond"
	UnitMole             = "hiveot:unit:mole"
	UnitPSI              = "hiveot:unit:psi"
	UnitPascal           = "hiveot:unit:pascal"
	UnitPercent          = "hiveot:unit:percent"
	UnitPound            = "hiveot:unit:pound"
	UnitPpm              = "hiveot:unit:ppm"
	UnitRadian           = "hiveot:unit:radian"
	UnitSecond           = "hiveot:unit:second"
	UnitVolt             = "hiveot:unit:volt"
	UnitWatt             = "hiveot:unit:watt"
)

// end of UnitClasses

// UnitClassesMap maps @type to symbol, title and description
var UnitClassesMap = map[string]struct {
	Symbol      string
	Title       string
	Description string
}{
	UnitHectoPascal:      {Symbol: "hPa", Title: "Hecto-Pascal", Description: "SI unit of atmospheric pressure. Equal to 100 pascal and 1 millibar."},
	UnitKilometerPerHour: {Symbol: "kph", Title: "Km per hour", Description: "Speed in kilometers per hour"},
	UnitAmpere:           {Symbol: "A", Title: "Ampere", Description: "Electrical current in Amperes based on the elementary charge flow per second"},
	UnitCelcius:          {Symbol: "°C", Title: "Celcius", Description: "Temperature in Celcius"},
	UnitKilowattHour:     {Symbol: "kWh", Title: "Kilowatt-hour", Description: "non-SI unit of energy equivalent to 3.6 megajoules."},
	UnitMilliBar:         {Symbol: "mbar", Title: "millibar", Description: "Metric unit of pressure. 1/1000th of a bar. Equal to 100 pascals. Amount of force it takes to move an object weighing a gram, one centimeter in one second."},
	UnitMilliSecond:      {Symbol: "ms", Title: "millisecond", Description: "Unit of time in milli-seconds. Equal to 1/1000 of a second."},
	UnitRadian:           {Symbol: "", Title: "Radian", Description: "Angle in 0-2pi"},
	UnitCandela:          {Symbol: "cd", Title: "Candela", Description: "SI unit of luminous intensity in a given direction. Roughly the same brightness as the common candle."},
	UnitFoot:             {Symbol: "ft", Title: "Foot", Description: "Imperial unit of distance. 1 foot equals 0.3048 meters"},
	UnitGallon:           {Symbol: "gl", Title: "Gallon", Description: "Unit of volume. 1 Imperial gallon is 4.54609 liters. 1 US liquid gallon is 3.78541 liters. 1 US dry gallon is 4.405 liters. "},
	UnitPascal:           {Symbol: "Pa", Title: "Pascal", Description: "SI unit of pressure. Equal to 1 newton of force applied over 1 square meter."},
	UnitPound:            {Symbol: "lbs", Title: "Pound", Description: "Imperial unit of weight. Equivalent to 0.453592 Kg. 1 Kg is 2.205 lbs"},
	UnitVolt:             {Symbol: "V", Title: "Volt", Description: "SI unit of electric potential; Energy consumption of 1 joule per electric charge of one coulomb"},
	UnitLiter:            {Symbol: "l", Title: "Liter", Description: "SI unit of volume equivalent to 1 cubic decimeter."},
	UnitMilliMeter:       {Symbol: "mm", Title: "Millimeter", Description: "Size in millimeter"},
	UnitMole:             {Symbol: "mol", Title: "Mole", Description: "SI unit of measurement for amount of substance. Eg, molecules."},
	UnitWatt:             {Symbol: "W", Title: "Watt", Description: "SI unit of power. Equal to 1 joule per second; or work performed when a current of 1 ampere flows across an electric potential of one volt."},
	UnitCount:            {Symbol: "(N)", Title: "Count", Description: ""},
	UnitMercury:          {Symbol: "Hg", Title: "Mercury", Description: "Unit of atmospheric pressure in the United States. 1 Hg equals 33.8639 mbar."},
	UnitMeterPerSecond:   {Symbol: "m/s", Title: "Meters per second", Description: "SI unit of speed in meters per second"},
	UnitFahrenheit:       {Symbol: "F", Title: "Fahrenheit", Description: "Temperature in Fahrenheit"},
	UnitKelvin:           {Symbol: "K", Title: "Kelvin", Description: "SI unit of thermodynamic temperature. 0 K represents absolute zero, the absence of all heat. 0 C equals +273.15K"},
	UnitPSI:              {Symbol: "PSI", Title: "PSI", Description: "Unit of pressure. Pounds of force per square inch. 1PSI equals 6984 Pascals."},
	UnitDegree:           {Symbol: "degree", Title: "Degree", Description: "Angle in 0-360 degrees"},
	UnitLumen:            {Symbol: "lm", Title: "Lumen", Description: "SI unit luminous flux. Measure of perceived power of visible light. 1lm = 1 cd steradian"},
	UnitMeter:            {Symbol: "m", Title: "Meter", Description: "Distance in meters. 1m=c/299792458"},
	UnitMilesPerHour:     {Symbol: "mph", Title: "Miles per hour", Description: "Speed in miles per hour"},
	UnitPercent:          {Symbol: "%", Title: "Percent", Description: "Fractions of 100"},
	UnitSecond:           {Symbol: "s", Title: "Second", Description: "SI unit of time based on caesium frequency"},
	UnitPpm:              {Symbol: "ppm", Title: "PPM", Description: "Parts per million"},
	UnitKilogram:         {Symbol: "kg", Title: "Kilogram", Description: ""},
	UnitLux:              {Symbol: "lx", Title: "Lux", Description: "SI unit illuminance. Equal to 1 lumen per square meter."},
}
