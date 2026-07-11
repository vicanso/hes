.PHONY: default test test-cover lint

test:
	go test -race -cover ./...

test-cover:
	go test -race -coverprofile=test.out ./... && go tool cover -html=test.out

lint:
	golangci-lint run
