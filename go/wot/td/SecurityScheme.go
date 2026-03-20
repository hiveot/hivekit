package td

// https://www.w3.org/TR/wot-thing-description11/#securityscheme
const (
	SecSchemeApiKey string = "apikey"
	SecSchemeAuto   string = "auto"
	SecSchemeBasic  string = "basic"
	SecSchemeBearer string = "bearer"
	SecSchemeDigest string = "digest"
	SecSchemePSK    string = "psk"
	SecSchemeNoSec  string = "nosec"
	SecSchemeOAuth2 string = "oauth2"
)

// Location of credentials in securityScheme
type CredLoc string

const (
	CredLocHeader CredLoc = "header"
	CredLocQuery  CredLoc = "query"
	CredLocBody   CredLoc = "body"
	CredLocCookie CredLoc = "cookie"
	CredLocAuto   CredLoc = "auto"
)

// SecurityScheme
type SecurityScheme struct {
	// JSON-LD keyword to label the object with semantic tags (or types).
	AtType []string `json:"@type,omitempty"`

	// Provides additional (human-readable) information based on a default language
	Description string `json:"description,omitempty"`

	// Can be used to support (human-readable) information in different languages. Also see MultiLanguage.
	Descriptions []string `json:"descriptions,omitempty"`

	// URI of the proxy server this security configuration provides access to.
	// If not given, the corresponding security configuration is for the endpoint.
	Proxy string `json:"proxy,omitempty"`

	// Identification of the security mechanism being configured.
	// See also SecSchemeXyz above.
	Scheme string `json:"scheme"`

	//--- fields for bearer schema
	// URI of the authorization server
	// Used in Bearer Schema
	Authorization string `json:"authorization,omitempty"`

	// Name for query, header, cookie or uri parameters
	// Used in Bearer Schema
	Name string `json:"name,omitempty"`

	// Encoding, encryption, or digest algorithm
	// eg: ES256, ES512-256
	// Used in Bearer Schema
	Alg string `json:"alg,omitempty"`

	// Specifies format of security authentication information
	// e.g.: jwt, cwt, jwe, jws
	// Used in Bearer Schema
	Format string `json:"format,omitempty"`

	// Specifies the location of security authentication information.
	// one of: header, query, body, cookie or auto
	In CredLoc `json:"in,omitempty"`
}
