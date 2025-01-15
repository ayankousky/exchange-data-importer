

lint:
	golangci-lint run

test:
	go test -v -cover ./...

test_race:
	go test -race -v -cover -timeout 5s ./...


.PHONY: lint test test_race