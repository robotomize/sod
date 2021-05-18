APP?=sod-srv
NAME?=sod server
RELEASE?=$(shell git describe --tags --abbrev=0)
GOOS?=linux
GOARCH=amd64
GO111MODULE?=on
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

unittest:
	go test -short $$(go list ./... | grep -v /vendor/)

test:
	go test -v -cover -covermode=atomic ./...

test-cover:
	go test -count=2 -race -timeout=10m ./... -coverprofile=coverage.out

.PHONY: build
build: clean
	GOARCH=${GOARCH} GO111MODULE=${GO111MODULE} CGO_ENABLED=0 GOOS=${GOOS} go build -o ${APP} -trimpath -ldflags "-s -w -X main.version=${RELEASE} -X main.buildTime=${BUILD_TIME} -X main.projectName=${NAME}" ./cmd/sod-srv

.PHONY: clean
clean:
	@rm -f ${APP}