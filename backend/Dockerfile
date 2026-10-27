FROM golang:1.27.3-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /api ./cmd/api && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /admin ./cmd/admin && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /maintenance ./cmd/maintenance
FROM build AS verification
RUN apk add --no-cache build-base
ENV CGO_ENABLED=1
CMD ["go", "test", "-race", "-v", "./..."]
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /api /api
COPY --from=build /admin /admin
COPY --from=build /maintenance /maintenance
EXPOSE 8080
ENTRYPOINT ["/api"]
