default: run

run:
	go run ./cmd/govern/...

test:
	go test --race ./...

format fmt:
	go fmt -x ./...
