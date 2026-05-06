FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/gateway-api ./apps/gateway-api/cmd/gateway-api

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/gateway-api /gateway-api
EXPOSE 8080
ENTRYPOINT ["/gateway-api"]

