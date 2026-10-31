FROM golang:latest AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /gluetun-hc ./cmd/gluetun-hc

FROM scratch
COPY --from=build /gluetun-hc /gluetun-hc
ENTRYPOINT ["/gluetun-hc"]
