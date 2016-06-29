default: clean dependencies build docker

clean:
	rm -rf web-proxy
dependencies:
	go get -u -v ./...
build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo  .
docker:
	docker build -t kazoup/web-proxy .
