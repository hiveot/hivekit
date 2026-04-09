package grpclib

import (
	"context"
	"fmt"
	"strings"

	"github.com/hiveot/hivekit/go/modules/transports"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// error result codes
var ErrMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
var ErrInvalidToken = status.Errorf(codes.PermissionDenied, "invalid token")

// GrpcAuthenticator adapter that adapts the transports.IAuthenticator interface to be used
// as a gRPC stream interceptor for authenticating incoming connections.
type GrpcAuthenticator struct {
	authenticator transports.IAuthenticator
}

// Determine the clientID and authenticate a new stream connection
func (srv *GrpcAuthenticator) Authenticate(ctx context.Context) (md metadata.MD, clientID string, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return md, "", ErrMissingMetadata
	}
	contextClientID := strings.Join(md[transports.ClientIDContextID], "")
	mdAuthn := md["authorization"]
	if len(mdAuthn) == 0 {
		return md, "", ErrInvalidToken
	}
	// remote the bearer prefix from the token
	token := mdAuthn[0]
	tokenParts := strings.Split(token, " ")
	bearerToken := tokenParts[1]
	clientID, _, err = srv.authenticator.ValidateToken(bearerToken)
	if err != nil {
		return md, "", err
	} else if clientID != contextClientID {
		return md, "", fmt.Errorf("Authenticate: Context clientID doesn't match token clientID")
	}
	return md, clientID, nil
}

func NewGrpcAuthenticator(authenticator transports.IAuthenticator) *GrpcAuthenticator {
	return &GrpcAuthenticator{
		authenticator: authenticator,
	}
}
