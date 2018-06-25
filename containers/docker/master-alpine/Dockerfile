FROM alpine:3.7

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 --branch release/1.8 https://github.com/gdachain/go-gdachain && \
  (cd go-gdaereum && make ggda) && \
  cp go-gdaereum/build/bin/ggda /ggda && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-gdaereum && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/ggda"]
