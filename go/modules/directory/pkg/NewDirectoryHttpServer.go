package directorypkg

import (
	"github.com/hiveot/hivekit/go/modules/directory"
	internal "github.com/hiveot/hivekit/go/modules/directory/internal/httpserver"
	"github.com/hiveot/hivekit/go/modules/transports"
)

func NewDirectoryHttpServer(httpServer transports.IHttpServer) directory.IDirectoryHttpServer {
	m := internal.StartDirectoryHttpServer(httpServer)
	return m
}
