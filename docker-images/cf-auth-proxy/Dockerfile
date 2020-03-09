ARG BASE_IMAGE=ubuntu:bionic
FROM $BASE_IMAGE as builder

RUN apt update && \
    apt install --no-install-recommends -y ca-certificates wget unzip && \
    update-ca-certificates && \
    apt-get clean

# Install Go. Note: loggers had 12.7
ARG GOLANG_SOURCE=https://dl.google.com/go/go1.13.6.linux-amd64.tar.gz
RUN wget -q $GOLANG_SOURCE -O go.tar.gz && \
    tar -xf go.tar.gz && \
    mv go /usr/local && \

#    should we do this?
#    rm go.tar.gz
ENV GOROOT=/usr/local/go
ENV GOPATH=$HOME/go
ENV GO111MODULE=on
ENV PATH=$GOROOT/bin:$GOPATH/bin:$PATH

ENV GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0

# copy/paste all the code into our repo or use log-cache's packages?
COPY / /log-cache-src/

RUN cd /log-cache-src && go build \
    -a \
    -installsuffix nocgo \
    -o /srv/cf-auth-proxy \
    -mod=vendor \
    ./cmd/cf-auth-proxy

# why all this builder-dpkg-list code?
RUN dpkg -l > /builder-dpkg-list

FROM $BASE_IMAGE

COPY --from=builder /builder-dpkg-list /builder-dpkg-list

# copies cf-auth-proxy code in order to run it
COPY --from=builder /srv/cf-auth-proxy /srv/cf-auth-proxy

# why does this user and group have to be created?
RUN groupadd --system cf-auth-proxy --gid 1000 && \
    useradd --no-log-init --system --gid cf-auth-proxy cf-auth-proxy --uid 1000

USER 1000:1000
WORKDIR /srv

#CAPI will be looking for 8080, so we'll have to change the internal metric-proxy code to be something else (8081 for ex)
ENV LOG_CACHE_GATEWAY_ADDR=metric-proxy:8081 \
    ADDR=:8080 \
    DISABLE_TLS_SERVER="true" \
    PROXY_CA_PATH="/ca/tls.crt" \
    METRICS_CA_FILE_PATH="/ca/tls.crt" \
    METRICS_CERT_FILE_PATH="/metrics-certs/tls.crt" \
    METRICS_KEY_FILE_PATH="/metrics-certs/tls.key" \
    INTERNAL_IP="127.0.0.1" \
    CAPI_ADDR="" \
    UAA_ADDR="" \
    UAA_CLIENT_ID="" \
    UAA_CLIENT_SECRET=""
#paths correct?

EXPOSE 8080
CMD [ "/srv/cf-auth-proxy" ]
