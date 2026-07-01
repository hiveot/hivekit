package internal

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hiveot/hivekit/go/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// error result codes
var ErrMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
var ErrInvalidToken = status.Errorf(codes.PermissionDenied, "invalid token")

// GrpcAuthenticator adapter that adapts the api.IAuthenticator interface to be used
// as a gRPC stream interceptor for authenticating incoming connections.
type GrpcAuthenticator struct {
	authenticator api.IAuthenticator
}

// Determine the clientID and authenticate a new stream connection
func (srv *GrpcAuthenticator) Authenticate(
	ctx context.Context) (md metadata.MD, clientID string, err error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return md, "", ErrMissingMetadata
	}

	p, ok := peer.FromContext(ctx)
	if ok {
		tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
		if !ok {
			return md, "", status.Errorf(codes.Internal, "connection is not using TLS")
		}
		// VerifiedChains[0] is the leaf certificate (the client's cert)
		if len(tlsInfo.State.PeerCertificates) > 0 {
			clientCert := tlsInfo.State.PeerCertificates[0]
			clientID := clientCert.Subject.CommonName
			// a valid certificate is found and the clientID is known
			return md, clientID, nil
		}
	}

	// no or invalid client cert, try auth token
	contextClientID := strings.Join(md[api.ClientIDContextID], "")
	mdAuthn := md["authorization"]
	if len(mdAuthn) == 0 {
		return md, "", ErrInvalidToken
	}
	// remote the bearer prefix from the token
	token := mdAuthn[0]
	tokenParts := strings.Split(token, " ")
	bearerToken := tokenParts[1]
	clientID, _, _, err = srv.authenticator.ValidateToken(bearerToken)
	if err != nil {
		slog.Warn("GrpcAuthenticator: auth failed: ", "err", err.Error())
		return md, "", err
	} else if clientID != contextClientID {
		return md, "", fmt.Errorf("Authenticate: Context clientID doesn't match token clientID")
	}
	return md, clientID, nil
}

func NewGrpcAuthenticator(authenticator api.IAuthenticator) *GrpcAuthenticator {
	return &GrpcAuthenticator{
		authenticator: authenticator,
	}
}
