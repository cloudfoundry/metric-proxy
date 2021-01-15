FROM golang:1.13 as builder

RUN mkdir /metric-proxy
WORKDIR /metric-proxy

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .
RUN make

FROM ubuntu:bionic

RUN apt-get update && \
    apt-get install --no-install-recommends -y curl && \
    apt-get clean

COPY --from=builder /metric-proxy/bin/metric-proxy-linux /bin/metric-proxy
USER 999:999

CMD ["/bin/metric-proxy"]
