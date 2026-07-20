# FreeZenith subdomain-registration service.
# Holds the Cloudflare token (via CLOUDFLARE_DNS_TOKEN); customer installers only
# receive a hostname back. Deploy behind TLS at register.freezenith.com.
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /register .

FROM gcr.io/distroless/static:nonroot
COPY --from=build /register /register
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/register"]
