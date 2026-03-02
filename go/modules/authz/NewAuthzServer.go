package authz

import (
	authzapi "github.com/hiveot/hivekit/go/modules/authz/api"
	authzserver "github.com/hiveot/hivekit/go/modules/authz/internal/server"
)

func NewAuthzServerModule(getRoleHandler func(clientID string) (role string, err error)) authzapi.IAuthzServer {
	m := authzserver.NewAuthzServer(getRoleHandler)
	return m
}
