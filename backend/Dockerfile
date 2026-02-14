FROM golang:1.22-alpine AS builder
WORKDIR /app
# Em redes com proxy/certificado corporativo, a verificação do sum.golang.org pode falhar
ENV GOSUMDB=off
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o /backend .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates wget
WORKDIR /app
COPY --from=builder /backend .
COPY migrations ./migrations
EXPOSE 8080
ENV PORT=8080
CMD ["./backend"]
