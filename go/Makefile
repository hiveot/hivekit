# Makefile to test HiveKit modules
DIST_FOLDER=./dist

.FORCE: examples

examples: counterdevice

counterdevice:
	mkdir -p $(DIST_FOLDER)
	go build -o $(DIST_FOLDER)/$@ examples/standalone/main.go

test: .FORCE
	go test -race -failfast -p 1 ./...
