build:
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo  -ldflags  -'w' -o keel .

image: build
	docker build -t karolisr/keel:0.0.1 -f Dockerfile .

push: 
	docker push karolisr/webhookrelay	