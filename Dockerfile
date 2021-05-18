FROM golang:1.16 AS builder

RUN apt-get -qq update && apt-get -yqq install upx

ENV GO111MODULE=on \
  CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=amd64

WORKDIR /src
COPY . .

RUN go build \
  -trimpath \
  -ldflags "-s -w -extldflags '-static'" \
  -installsuffix cgo \
  -o /bin/sod \
  ./cmd/sod-srv

RUN strip /bin/sod
RUN upx -q -9 /bin/sod


RUN mkdir /data

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/sod /bin/sod
COPY --from=builder /data /data

VOLUME /data

ENTRYPOINT ["/bin/sod"]