test:
	go test -v -count=1 -race ./...

lint:
	golangci-lint run ./...

build:
	GOOS=linux GOARCH=arm64 go build -o smart-meter-exporter
