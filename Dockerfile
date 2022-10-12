FROM golang:1.19 AS builder
WORKDIR /go/src 
ENV GO111MODULE=on GOPROXY=https://goproxy.cn CGO_ENABLED=0   
ADD . .
RUN go mod tidy && go build -tags netgo -o /go/bin/repojob cmd/main.go

FROM alpine:3.10
COPY --from=builder /go/bin/repojob /
ENTRYPOINT ["/repojob"]