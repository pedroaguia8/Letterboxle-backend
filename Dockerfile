FROM golang:1.26

WORKDIR /usr/src/app

RUN go install github.com/pressly/goose/v3/cmd/goose@v3.27.3

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o /usr/local/bin/app .

CMD ["app"]
