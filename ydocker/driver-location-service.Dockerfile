FROM golang:1.24-alpine

WORKDIR /app

COPY . .

RUN go build -o main ./cmd/ride-hail/main.go

EXPOSE 3000

CMD ["./main", "--mode=dls"]