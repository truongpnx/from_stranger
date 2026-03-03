FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download

COPY . .
RUN mkdir -p /src/dist && go build -o /src/dist/server ./cmd/server

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /src/dist/server /app/server
COPY --from=builder /src/internal/app/templates /app/internal/app/templates
COPY --from=builder /src/internal/static /app/internal/static

EXPOSE 8080
ENV PORT=8080

CMD ["/app/server"]
