FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod main.go ./
RUN go build -o dinq-latex .

FROM alpine:3.20

# Install tectonic
RUN apk add --no-cache tectonic

COPY --from=builder /app/dinq-latex /usr/local/bin/dinq-latex

EXPOSE 8092
CMD ["dinq-latex"]
