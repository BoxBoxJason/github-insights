FROM docker.io/library/golang:1.27.1-alpine@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS build

WORKDIR /app

COPY . .

ENV GO111MODULE=on \
    CGO_ENABLED=0

RUN apk add --no-cache make git && \
  make build

FROM docker.io/library/alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS security_provider

RUN addgroup -S github-insights \
    && adduser -S github-insights -G github-insights

FROM scratch

COPY --from=security_provider /etc/passwd /etc/passwd

USER github-insights

COPY --from=build /app/bin/github-insights /usr/local/bin/github-insights

ENTRYPOINT [ "/usr/local/bin/github-insights" ]
