build:
	go build -o bin/lanbox ./cmd/lanbox

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l .
