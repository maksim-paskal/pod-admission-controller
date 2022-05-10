FROM alpine:latest

WORKDIR /app/

RUN apk upgrade \
&& addgroup -g 101 -S app \
&& adduser -u 101 -D -S -G app app

COPY ./pod-admission-controller /app/pod-admission-controller

USER 101

ENTRYPOINT [ "/app/pod-admission-controller" ]