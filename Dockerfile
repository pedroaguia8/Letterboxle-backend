FROM golang:1.27.2@sha256:5bc7f572bbaa98885a3a1fd9c0aa76b59e3e14e8628bfc316bbfd0c701e4818c AS build

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# static binary (no libc), so it runs on distroless/static
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app .

# CA certificates, tzdata and a nonroot user; no shell or package manager
FROM gcr.io/distroless/static-debian13:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3

COPY --from=build /out/app /usr/local/bin/app

CMD ["app"]
