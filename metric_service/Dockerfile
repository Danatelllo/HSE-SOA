FROM golang:alpine

WORKDIR /metric_service

COPY . .

RUN go mod download

RUN go build -o server main.go

CMD ["./server"]
