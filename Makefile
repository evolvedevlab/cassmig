APP_NAME=cassmig

build:
	@echo "Building $(APP_NAME)"
	@mkdir -p dist

	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(APP_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(APP_NAME)-darwin-amd64 .

	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(APP_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(APP_NAME)-linux-arm64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags="-s -w" -o dist/$(APP_NAME)-linux-386 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -ldflags="-s -w" -o dist/$(APP_NAME)-linux-arm .

	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(APP_NAME)-windows-amd64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(APP_NAME)-windows-arm64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags="-s -w" -o dist/$(APP_NAME)-windows-386.exe .
