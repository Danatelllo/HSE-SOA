FROM golang:alpine

WORKDIR /rest_api_server

COPY . .

RUN go mod download

RUN go build -o server main.go

CMD ["./server"]
