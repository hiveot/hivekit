package api

// helper for building a login request message
type UserLoginArgs struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
