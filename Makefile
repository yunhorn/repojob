bin: 
	@echo "begin build bin"
	@go mod tidy && go build -tags netgo -o repojob cmd/main.go

build-ghcr:
	docker build -t ghcr.io/yunhorn/repojob:v0.1.0 .
	docker push ghcr.io/yunhorn/repojob:v0.1.0
