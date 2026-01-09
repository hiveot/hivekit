package authnapi

// helper for building a login request message
// tbd this should probably go elsewhere.
type UserLoginArgs struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
