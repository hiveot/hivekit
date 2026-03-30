package grpcserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/hiveot/hivekit/go/modules/transports"
	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"google.golang.org/grpc/metadata"
)

type GrpcAuthenticator struct {
	authenticator transports.IAuthenticator
}

// Determine the clientID and authenticate a new stream connection
func (srv *GrpcAuthenticator) Authenticate(ctx context.Context) (md metadata.MD, clientID string, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return md, "", grpcapi.ErrMissingMetadata
	}
	contextClientID := strings.Join(md[transports.ClientIDContextID], "")
	mdAuthn := md["authorization"]
	if len(mdAuthn) == 0 {
		return md, "", grpcapi.ErrInvalidToken
	}
	token := mdAuthn[0] // FIXME: is this bearer???
	clientID, _, err = srv.authenticator.ValidateToken(token)
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
