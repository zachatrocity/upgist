FROM rust:alpine as builder
RUN apk add --no-cache musl-dev openssl-dev git

WORKDIR /usr/src/upgist
COPY . .

RUN cargo build --release

FROM alpine:latest
RUN apk add --no-cache git openssh-client

WORKDIR /app
COPY --from=builder /usr/src/upgist/target/release/upgist /usr/local/bin/
COPY --from=builder /usr/src/upgist/static /app/static

EXPOSE 3000
CMD ["upgist"]
