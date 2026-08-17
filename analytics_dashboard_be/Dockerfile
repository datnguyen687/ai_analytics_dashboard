# ---- build ----
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Static binary so it runs on a minimal base.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/analytics .

# ---- runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget && \
    wget -qO /usr/local/bin/dbmate \
      https://github.com/amacneil/dbmate/releases/download/v2.21.0/dbmate-linux-amd64 && \
    chmod +x /usr/local/bin/dbmate
WORKDIR /app
COPY --from=build /out/analytics /app/analytics
COPY db/migrations /app/db/migrations
COPY data /app/data
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh
EXPOSE 8080
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["serve"]
