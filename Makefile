.PHONY: run build test clean

# Run the server
run:
	go run cmd/server/main.go

# Build the binary
build:
	go build -o tracky cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f tracky

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
	