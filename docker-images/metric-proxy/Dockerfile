ARG BASE_IMAGE=ubuntu:bionic
FROM $BASE_IMAGE as builder

RUN apt update && \
    apt install --no-install-recommends -y make ca-certificates wget zip unzip && \
    update-ca-certificates && \
    apt-get clean

# Install Go
ARG GOLANG_SOURCE=https://dl.google.com/go/go1.13.6.linux-amd64.tar.gz
RUN wget -q $GOLANG_SOURCE -O go.tar.gz && \
    tar -xf go.tar.gz && \
    mv go /usr/local
ENV GOROOT=/usr/local/go
ENV GOPATH=$HOME/go
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH

ENV GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0


COPY / /metric-proxy/

RUN cd /metric-proxy && go build \
    -a \
    -installsuffix nocgo \
    -o /bin/metric-proxy \
    -mod=readonly \
    .

FROM $BASE_IMAGE

RUN apt update && \
    apt install --no-install-recommends -y ca-certificates && \
    update-ca-certificates && \
    apt-get clean

COPY --from=builder /bin/metric-proxy /bin/metric-proxy
USER 999:999

CMD ["/bin/metric-proxy"]
