package authz

import (
	authzapi "github.com/hiveot/hivekit/go/modules/authz/api"
	"github.com/hiveot/hivekit/go/modules/authz/internal"
)

func NewAuthzService(getRoleHandler func(clientID string) (role string, err error)) authzapi.IAuthzService {
	m := internal.NewAuthzService(getRoleHandler)
	return m
}
