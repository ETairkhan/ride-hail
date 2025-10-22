FROM golang:1.23

WORKDIR /cmd/ride-hail

COPY go.mod go.sum ./
RUN go mod download

COPY . . 

RUN go build -o app ./cmd/ride-hail/main.go

EXPOSE 8080

CMD ["./app"]