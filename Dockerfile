FROM golang:1.26.10@sha256:670bb5aa5349ae03b8516764718486f4f16399134bdc9acd0c46a23f043ab05c AS build

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# static binary (no libc), so it runs on distroless/static
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app .

# CA certificates, tzdata and a nonroot user; no shell or package manager
FROM gcr.io/distroless/static-debian13:nonroot@sha256:18f4cc26c9ffa7f20a359298be7deadac5c607c8e03a02b8119f1a5fc2f82c99

COPY --from=build /out/app /usr/local/bin/app

CMD ["app"]
