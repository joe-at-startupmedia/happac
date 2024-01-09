GO ?= go
GOFMT ?= gofmt "-s"
GOFILES := $(shell find . -name "*.go")
 
all: build

build:
	$(GO) mod tidy
	$(GO) build -o bin/happac happac.go

install:
	cp bin/happac /usr/local/bin/happac

fmt:
	$(GOFMT) -w $(GOFILES)
