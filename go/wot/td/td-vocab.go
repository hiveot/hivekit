// Package td with WoT TD vocabulary
package td

// HiveOT operations that are not defined with the TD and used in HiveOT.
const (
	HTOpPing              = "ping"
	HTOpLogout            = "logout"
	HTOpObserveAction     = "observeaction"
	HTOpObserveAllActions = "observeallactions"
	HTOpReadEvent         = "readevent"
	HTOpReadAllEvents     = "readallevents"
)

// See https://www.w3.org/TR/2020/WD-wot-thing-description11-20201124/#sec-core-vocabulary-definition

// WoT TD1.1 operations
const (
	// actions
	OpCancelAction    = "cancelaction"
	OpInvokeAction    = "invokeaction"
	OpQueryAction     = "queryaction"
	OpQueryAllActions = "queryallactions"
	// events
	OpSubscribeAllEvents   = "subscribeallevents"
	OpSubscribeEvent       = "subscribeevent"
	OpUnsubscribeAllEvents = "unsubscribeallevents"
	OpUnsubscribeEvent     = "unsubscribeevent"
	// properties
	OpObserveAllProperties        = "observeallproperties"
	OpObserveMultipleProperties   = "observemultipleproperties"
	OpObserveProperty             = "observeproperty"
	OpReadAllProperties           = "readallproperties"
	OpReadMultipleProperties      = "readmultipleproperties"
	OpReadProperty                = "readproperty"
	OpUnobserveAllProperties      = "unobserveallproperties"
	OpUnobserveMultipleProperties = "unobservemultipleproperties"
	OpUnobserveProperty           = "unobserveproperty"
	OpWriteMultipleProperties     = "writemultipleproperties"
	OpWriteProperty               = "writeproperty"
)

// WoT data types
const (
	WoTDataType         = "type"
	DataTypeAnyURI      = "anyURI"
	DataTypeArray       = "array"
	DataTypeBool        = "boolean"
	DataTypeDateTime    = "dateTime"
	DataTypeInteger     = "integer"
	DataTypeNone        = ""
	DataTypeNumber      = "number"
	DataTypeObject      = "object"
	DataTypeString      = "string"
	DataTypeUnsignedInt = "unsignedInt"
)

// TD-1.1 affordance and data schema vocabulary
const (
	WoTActions              = "actions"
	WoTDescription          = "description"
	WoTDescriptions         = "descriptions"
	WoTDigestSecurityScheme = "DigestSecurityScheme"
	WoTEnum                 = "enum"
	WoTEvents               = "events"
	WoTFormat               = "format"
	WoTForms                = "forms"
	WoTHref                 = "href"
	WoTID                   = "id"
	WoTInput                = "input"
	WoTLinks                = "links"
	WoTMaxItems             = "maxItems"
	WoTMaxLength            = "maxLength"
	WoTMaximum              = "maximum"
	WoTMinItems             = "minItems"
	WoTMinLength            = "minLength"
	WoTMinimum              = "minimum"
	WoTModified             = "modified"
	WoTNoSecurityScheme     = "NoSecurityScheme"
	WoTOAuth2SecurityScheme = "OAuth2SecurityScheme"
	WoTOperation            = "op"
	WoTProperties           = "properties"
	WoTReadOnly             = "readOnly"
	WoTRequired             = "required"
	WoTSecurity             = "security"
	WoTSupport              = "support"
	WoTTitle                = "title"
	WoTTitles               = "titles"
	WoTVersion              = "version"
)
