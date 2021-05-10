FROM golang:1.16 as build

RUN mkdir /workspace
WORKDIR /workspace

ARG VERSION
RUN test -n "$VERSION"

RUN git clone https://github.com/kubernetes-sigs/metrics-server.git --branch $VERSION --depth=1 /go/src/github.com/kubernetes-incubator/metrics-server && \
    cd /go/src/github.com/kubernetes-incubator/metrics-server && \
    sed -i'' 's/\_output\/\$\*\/metrics-server/\/workspace\/metrics-server/g' Makefile && \
    make all

FROM ubuntu:bionic
COPY --from=build /workspace/metrics-server /

ENTRYPOINT ["/metrics-server"]
