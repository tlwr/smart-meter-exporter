test:
	ginkgo ./...

lint:
	golangci-lint run ./...

build:
	GOOS=linux GOARCH=arm64 go build -o smart-meter-exporter
