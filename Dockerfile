FROM golang:1.13 as builder

COPY / /metric-proxy/

RUN cd /metric-proxy && go build \
    -a \
    -installsuffix nocgo \
    -o /bin/metric-proxy \
    -mod=readonly \
    .

FROM ubuntu:bionic

RUN apt-get update && \
    apt-get install --no-install-recommends -y curl && \
    apt-get clean

COPY --from=builder /bin/metric-proxy /bin/metric-proxy
USER 999:999

CMD ["/bin/metric-proxy"]
