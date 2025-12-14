build:
	GOOS=windows GOARCH=amd64 go build -o bin/kongi.exe
	GOOS=darwin GOARCH=amd64 go build -o bin/kongi
	GOOS=linux GOARCH=amd64 go build -o bin/kongi

test: 
	go test -v ./...
	
run:
	go run main.go