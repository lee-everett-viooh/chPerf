.PHONY: build run generate dev

# Generate templ files
generate:
	templ generate

# Build the application
build: generate
	go build -o chperf .

# Run the application
run: build
	./chperf

# Development: generate and run
dev: generate
	go run .
