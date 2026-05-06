FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/notifier-service ./apps/notifier-service/cmd/notifier-service

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/notifier-service /notifier-service
EXPOSE 9092
ENTRYPOINT ["/notifier-service"]

