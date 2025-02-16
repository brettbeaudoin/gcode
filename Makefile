# Makefile for building the G-code Modifier utility

BINARY_NAME = gcode_modifier

all: help

build:
	@echo "Building the Go executable..."
	go build -o ./bin/$(BINARY_NAME) gcode_modifier.go
	@echo "Build complete. Run './bin/$(BINARY_NAME)' to execute."

clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	@echo "Clean complete."

help:
	@cat Makefile
