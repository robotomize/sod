APP?=sod
NAME?=sod
RELEASE?=0.0.0
GOOS?=linux
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

unittest:
	go test -short $$(go list ./... | grep -v /vendor/)

test:
	go test -v -cover -covermode=atomic ./...

.PHONY: build
build: clean
# build server
	CGO_ENABLED=0 GOOS=${GOOS} go build ./cmd/sod \
		-ldflags "-X main.version=${RELEASE}  -X main.buildTime=${BUILD_TIME} -X main.name=${NAME}" \
		-o build/${APP}

.PHONY: clean
clean:
	@rm -f build/${APP}