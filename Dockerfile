FROM golang:1.27.3@sha256:e519d3b0a5b285e5aa53ab9eef02bc6bbb9cb04608389b743a136a9ce75f0d1e AS build

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# static binary (no libc), so it runs on distroless/static
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app .

# CA certificates, tzdata and a nonroot user; no shell or package manager
FROM gcr.io/distroless/static-debian13:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329

COPY --from=build /out/app /usr/local/bin/app

CMD ["app"]
