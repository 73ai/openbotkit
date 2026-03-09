FROM golang:1.25-alpine AS builder

ARG TARGETOS TARGETARCH

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /obk .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata sqlite
COPY --from=builder /obk /usr/local/bin/obk

RUN adduser -D -h /home/obk obk && \
    mkdir -p /data/gmail /data/whatsapp /data/history /data/user_memory /data/applenotes && \
    chown -R obk:obk /data
USER obk
WORKDIR /home/obk

EXPOSE 8443

ENTRYPOINT ["obk"]
CMD ["server"]
