FROM node:14-alpine

COPY --from=golang:1.17-alpine /usr/local/go/ /usr/local/go/

ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /home/KeyValueDB/

RUN apk add --update redis

RUN apk --no-cache add ca-certificates

COPY ./ ./

CMD ["go","run", "main.go"]