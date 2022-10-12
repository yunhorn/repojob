bin: 
	@echo "begin build bin"
	@go mod tidy && go build -tags netgo -o repojob cmd/main.go

build-ghcr:
	docker build -t ghcr.io/yunhorn/repojob:v0.2.0 .
	docker push ghcr.io/yunhorn/repojob:v0.2.0

build-latest:
	docker build -t ghcr.io/yunhorn/repojob:latest .
	docker push ghcr.io/yunhorn/repojob:latest
