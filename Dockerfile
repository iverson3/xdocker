ARG SYSTEMD="false"
ARG GO_VERSION=1.0.1
ARG BASE_DEBIAN_DISTRO="alpine"
ARG GOLANG_IMAGE="${BASE_DEBIAN_DISTRO}-golang@v${GO_VERSION}"

FROM ${GOLANG_IMAGE} AS builder

ENV PATH /usr/xxx:$PATH
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /usr/go/src/test/

COPY . .

RUN go build -tags netgo -o goserver main.go

FROM alpine

COPY --from=builder /usr/go/src/test/goserver /bin/goserver

COPY --from=builder /usr/go/src/test/source /usr/local/resources

ENTRYPOINT ["/bin/goserver", "param1", "param2"]
