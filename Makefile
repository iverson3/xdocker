APP=xdocker

.PHONY: build

help:
	@echo "usage: make <option>"
	@echo "options and effects:"
	@echo "    help   : Show help"
	@echo "    build  : Build the binary of this project for current platform"

build:
	@go build -o ${APP}