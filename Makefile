default: clean dependencies build docker

clean:
	rm -rf web-proxy
dependencies:
	go get -d -v -t  ./...
build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo  .
docker:
	docker build -t kazoup/web-proxy .
deploy:
	docker push kazoup/web-proxy
