# Build context: reminder/ (app separado)
# Railway: Root Directory = "reminder", Dockerfile Path = "Dockerfile"
FROM golang:1.22-alpine AS builder
WORKDIR /app
ENV GOSUMDB=off
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o /reminder .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /reminder .
COPY --from=builder /app/migrations ./migrations
EXPOSE 8081
ENV PORT=8081
CMD ["./reminder"]
