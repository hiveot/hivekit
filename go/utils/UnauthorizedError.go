package utils

var UnauthorizedError error = unauthorizedError{}

// UnauthorizedError for dealing with authorization problems
type unauthorizedError struct {
	Message string
}

func (e unauthorizedError) Error() string {
	return "Unauthorized: " + e.Message
}
