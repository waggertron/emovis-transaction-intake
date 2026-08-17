FROM golang:1.26.6-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/transaction-service ./cmd/transaction-service
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/topic-bootstrap ./cmd/topic-bootstrap

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
USER 65532:65532
COPY --from=build /out/transaction-service /transaction-service
COPY --from=build /out/topic-bootstrap /topic-bootstrap

FROM runtime AS api
EXPOSE 8080
ENTRYPOINT ["/transaction-service"]
CMD ["api"]

FROM runtime AS worker
ENTRYPOINT ["/transaction-service"]
CMD ["worker"]

FROM runtime AS local
EXPOSE 8080
ENTRYPOINT ["/transaction-service"]
CMD ["local"]

FROM runtime AS topic-bootstrap
ENTRYPOINT ["/topic-bootstrap"]
