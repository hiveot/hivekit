# Makefile to test HiveKit cells
DIST_FOLDER=./dist
CACHE_OPT="-count=1"


.FORCE: 


test: .FORCE
	go test ${CACHE_OPT} -race -failfast -p 1 ./...
