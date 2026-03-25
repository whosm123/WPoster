.PHONY: build clean test run

build:
	go build -o bin/wposter main.go

clean:
	rm -rf bin/

test:
	go test ./...

run: build
	./bin/wposter

install: build
	cp bin/wposter /usr/local/bin/

dev:
	go run main.go