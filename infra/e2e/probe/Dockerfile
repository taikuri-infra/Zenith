# e2e-probe — the sample app the E2E suite deploys (see main.go).
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /probe .

FROM gcr.io/distroless/static:nonroot
COPY --from=build /probe /probe
EXPOSE 8080
ENTRYPOINT ["/probe"]
