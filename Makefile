releasebuild: checkem.go
	go build -ldflags="-s -w"

install: releasebuild
	sudo cp ./checkem /usr/local/bin/ce

uninstall:
	sudo rm /usr/local/bin/ce

clean:
	go clean
