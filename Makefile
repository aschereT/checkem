releasebuild: checkem.go
	go build -ldflags="-s -w"

install: releasebuild
	sudo cp ./checkem /usr/local/bin

clean:
	go clean