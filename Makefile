
build:
	clear
	@echo "building voyager"
	@go build ./...

update:
	clear
	@echo "updating dependencies..."
	@go get -u -t ./...
	@go mod tidy 

test:
	@clear 
	@echo "testing..."
	@go test ./...
