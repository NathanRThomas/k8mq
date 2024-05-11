
# go params
GOCMD=go
GOBUILD=$(GOCMD) build -buildvcs=false
GOTEST=$(GOCMD) test -v -run
GOPATH=/usr/local/bin
DIR=$(shell pwd)

build: build-server 
build: build-client 

build-server:
	@echo "building k8mq server"
	@$(GOBUILD) -o $(GOPATH)/k8mq-server ./server/cmd/

build-client:
	@echo "building k8mq client"
	@$(GOBUILD) ./client/...

update:
	clear
	@echo "updating dependencies..."
	@go get -u -t ./...
	@go mod tidy 

test:
	@clear 
	@echo "testing QA..."
	@$(GOTEST) QA ./...

test-net:
	@clear 
	@echo "testing NET..."
	@$(GOTEST) Net ./...
