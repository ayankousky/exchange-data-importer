

lint:
	golangci-lint run

test:
	go test -cover ./...

rtest:
	go test -race -cover -timeout 5s ./...

gen:
	go generate ./...


.PHONY: lint test rtest gen