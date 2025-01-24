

lint:
	golangci-lint run

test:
	go test -cover ./...

test_race:
	go test -race -cover -timeout 5s ./...


.PHONY: lint test test_race