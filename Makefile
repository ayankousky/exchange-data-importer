

lint:
	golangci-lint run

test:
	go test -cover ./...

rtest:
	go test -race -cover -timeout 5s ./...


.PHONY: lint test rtest