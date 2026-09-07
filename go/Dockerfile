# Build the Go server, then run it on distroless.
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/guardianforge ./cmd/guardianforge

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/guardianforge /guardianforge
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/guardianforge"]
