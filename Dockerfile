FROM golang:1.9-alpine
RUN mkdir -p /app
ADD auroramysql-preprovision.go  /app/auroramysql-preprovision.go
ADD build.sh /app/build.sh
RUN chmod +x /app/build.sh
RUN apk add --no-cache git \
    && /app/build.sh \
    && apk del git
CMD ["/app/auroramysql-preprovision"]