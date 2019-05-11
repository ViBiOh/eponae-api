FROM golang:1.12 as builder

WORKDIR /app
COPY . .

RUN make eponae-api \
 && curl -s -o /app/cacert.pem https://curl.haxx.se/ca/cacert.pem

FROM scratch

EXPOSE 1080

HEALTHCHECK --retries=10 CMD [ "/api", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/api" ]

ARG APP_VERSION
ENV VERSION=${APP_VERSION}

COPY ./doc /doc
COPY --from=builder /app/cacert.pem /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/bin/eponae-api /api
