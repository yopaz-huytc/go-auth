FROM golang:1.22.3-alpine3.20

WORKDIR /usr/src/app

RUN go install github.com/cosmtrek/air@latest

COPY . .

RUN go mod tidy

CMD ["air", "-c", ".air.toml"]
