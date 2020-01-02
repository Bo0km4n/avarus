build:
	go build .
clean:
	rm -rf ./output
.PHONY: release
release:
	GOOS=linux GOARCH=amd64 go build -o bin/avarus_linux_amd64
	GOOS=darwin GOARCH=amd64 go build -o bin/avarus_darwin_amd64
	GOOS=windows GOARCH=amd64 go build -o bin/avarus_windows_amd64
