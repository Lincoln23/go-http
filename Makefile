# References:
# https://stackoverflow.com/questions/2145590/what-is-the-purpose-of-phony-in-a-makefile

.PHONY: all
all: run

.PHONY: bin/go-http
bin/go-http:
	go build -v -o $@ ./cmd/go-http

.PHONY: run
run: bin/go-http
	./bin/go-http
