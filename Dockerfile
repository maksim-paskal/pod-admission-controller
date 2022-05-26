FROM alpine:latest

WORKDIR /app/

RUN apk upgrade \
&& addgroup -g 30626 -S app \
&& adduser -u 30626 -D -S -G app app

COPY ./pod-admission-controller /app/pod-admission-controller

USER 30626

ENTRYPOINT [ "/app/pod-admission-controller" ]