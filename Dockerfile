FROM --platform=$BUILDPLATFORM golang:latest AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /gluetun-hc ./cmd/gluetun-hc

FROM scratch
COPY --from=build /gluetun-hc /gluetun-hc
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 CMD ["/gluetun-hc", "healthcheck"]
USER 65534:65534
ENTRYPOINT ["/gluetun-hc"]
