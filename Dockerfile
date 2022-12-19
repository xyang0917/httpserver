FROM golang:1.18.1 AS build
WORKDIR /httpserver/
COPY . .
ENV CGO_ENABLED=0
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct
RUN GOOS=linux GOARCH=amd64 go build -installsuffix cgo -o httpserver main.go

FROM busybox
COPY --from=build /httpserver/httpserver /httpserver/httpserver
ENV ENV=local SERVER_PORT=8080
EXPOSE ${SERVER_PORT}

WORKDIR /httpserver/
ENTRYPOINT ["./httpserver"]