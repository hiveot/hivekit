# Makefile for building examples
DIST_FOLDER=./dist
# optimize binary. Reduces size by about 30%.
# to reduce further, checkout 'upx'
OPTS=-ldflags="-s -w" -trimpath
CGO="0"

.FORCE: 

all: appenv sadevice cli tui gateway rcdevice

# show app environment being used
appenv: .FORCE
	CGO_ENABLED=$(CGO) go build $(OPTS) -o $(DIST_FOLDER)/$@ $@/main.go

# stand-alone device with server
sadevice: .FORCE
	CGO_ENABLED=$(CGO) go build $(OPTS) -o $(DIST_FOLDER)/$@ $@/main.go

# CLI for exploring devices
cli: .FORCE
	CGO_ENABLED=$(CGO) go build $(OPTS) -o $(DIST_FOLDER)/$@ $@/main.go

# text-ui for exploring devices
tui: .FORCE
	CGO_ENABLED=$(CGO) go build $(OPTS) -o $(DIST_FOLDER)/$@ $@/main.go

# gateway with servers, auth and router  
gateway: .FORCE
	CGO_ENABLED=$(CGO) go build $(OPTS) -o $(DIST_FOLDER)/$@ $@/main.go

# reverse connection device, connecting to the gateway
rcdevice: .FORCE
	CGO_ENABLED=$(CGO) go build $(OPTS) -o $(DIST_FOLDER)/$@ $@/main.go
