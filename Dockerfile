FROM golang:1.25-alpine AS builder
ARG VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN echo "$VERSION" > VERSION && CGO_ENABLED=0 go build -ldflags="-s -w" -o /ssh-to-go .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tmux openssh-client
COPY --from=builder /ssh-to-go /usr/local/bin/ssh-to-go

# Config: hosts, listen address, poll interval
VOLUME /etc/ssh-to-go
# Data: settings.json, keys/ (keypairs + index)
VOLUME /data

COPY docker-config.yaml /etc/ssh-to-go/config.yaml

EXPOSE 8080
ENTRYPOINT ["ssh-to-go"]
CMD ["-config", "/etc/ssh-to-go/config.yaml"]
