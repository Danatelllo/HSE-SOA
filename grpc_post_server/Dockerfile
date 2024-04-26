FROM golang:alpine

WORKDIR /grpc_post_server

COPY . .

RUN go mod download

RUN go build -o server main.go

CMD ["./server"]
