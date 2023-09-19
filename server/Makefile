# go params
GOCMD=go
GOBUILD=$(GOCMD) build -buildvcs=false
GOTEST=$(GOCMD) test -v -run
GOPATH=/usr/local/bin
DIR=$(shell pwd)

build:
	clear
	@echo "building k8mq server"
	@$(GOBUILD) -o $(GOPATH)/k8mq-server ./main/

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

run: build
run:
	@k8mq-server -p 5555 -vv
