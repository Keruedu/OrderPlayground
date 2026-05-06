FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/inventory-service ./apps/inventory-service/cmd/inventory-service

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/inventory-service /inventory-service
EXPOSE 9091
ENTRYPOINT ["/inventory-service"]

