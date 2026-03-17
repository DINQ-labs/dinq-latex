FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod main.go ./
RUN go build -o dinq-latex .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates curl libreoffice \
    && curl -fsSL https://github.com/tectonic-typesetting/tectonic/releases/download/tectonic%400.15.0/tectonic-0.15.0-x86_64-unknown-linux-musl.tar.gz \
       | tar xz -C /usr/local/bin/ \
    && chmod +x /usr/local/bin/tectonic \
    && apk del curl

COPY --from=builder /app/dinq-latex /usr/local/bin/dinq-latex

EXPOSE 8092
CMD ["dinq-latex"]
