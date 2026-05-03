FROM docker.io/library/golang:1.26.2-alpine AS build

WORKDIR /app

COPY . .

ENV GO111MODULE=on \
    CGO_ENABLED=0

RUN apk add --no-cache make git && \
  make build

FROM docker.io/library/alpine:3.23.4 AS security_provider

RUN addgroup -S github-insights \
    && adduser -S github-insights -G github-insights

FROM scratch

COPY --from=security_provider /etc/passwd /etc/passwd

USER github-insights

COPY --from=build /app/bin/github-insights /usr/local/bin/github-insights

ENTRYPOINT [ "/usr/local/bin/github-insights" ]
