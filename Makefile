build:
	go build -o bin/happac agent.go

install:
	cp bin/happac /usr/local/bin/happac

all: build
