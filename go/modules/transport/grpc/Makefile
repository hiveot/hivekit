
# protobuffer compilation
# Makefile for the protobuf compilation
GOOPT=--go_opt=paths=source_relative
GOOUT=--go_out=./api
GRPC=--go-grpc_out=./api --go-grpc_opt=paths=source_relative

PROTOC=protoc --proto_path=./api $(GOOPT) $(GOOUT) $(GRPC)


grpc: ## Compile the grpc module
	$(PROTOC) api/grpc_transport.proto