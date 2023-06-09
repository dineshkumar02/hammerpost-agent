# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
BINARY_NAME=hammerpost-agent
COMMIT=$(shell git rev-parse --short HEAD)
DATE=$(shell git log -1 --format=%ci)
Version=0.1.0

all: build
build: 
		GOOS=linux CGO_ENABLED=0 $(GOBUILD) -o $(BINARY_NAME) -v -ldflags="-X 'main.Version=${Version}' -X 'main.GitCommit=${COMMIT}' -X 'main.CommitDate=${DATE}'"
test: 
		$(GOTEST) -v ./...
clean: 
		$(GOCLEAN)
		rm -f $(BINARY_NAME)
