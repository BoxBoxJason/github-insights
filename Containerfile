FROM docker.io/library/golang:1.27.3-alpine@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18 AS build

WORKDIR /app

COPY . .

ENV GO111MODULE=on \
    CGO_ENABLED=0

RUN apk add --no-cache make git && \
  make build

FROM docker.io/library/alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a AS security_provider

RUN addgroup -S github-insights \
    && adduser -S github-insights -G github-insights

FROM scratch

COPY --from=security_provider /etc/passwd /etc/passwd

USER github-insights

COPY --from=build /app/bin/github-insights /usr/local/bin/github-insights

ENTRYPOINT [ "/usr/local/bin/github-insights" ]
