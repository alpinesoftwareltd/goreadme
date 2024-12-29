GOOS=windows GOARCH=arm64 go build -o bin/goreadme-windows-arm64 .
GOOS=windows GOARCH=amd64 go build -o bin/goreadme-windows-amd64 .

GOOS=linux GOARCH=arm64 go build -o bin/goreadme-linux-arm64 .
GOOS=linux GOARCH=amd64 go build -o bin/goreadme-linux-amd64 .

GOOS=darwin GOARCH=arm64 go build -o bin/goreadme-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -o bin/goreadme-darwin-amd64 .
