FROM golang:1.19-alpine as builder

# install build dependencies
RUN apk add --no-cache --update git gcc musl-dev

ARG LNTOP_SRC_PATH

WORKDIR $GOPATH/src/github.com/hieblmi/lntop
COPY "$LNTOP_SRC_PATH" .

ENV GO111MODULE=on
RUN go install ./...

# ---------------------------------------------------------------------------------------------------------------------------

FROM golang:1.19-alpine as final

RUN apk add --no-cache \
    bash fish \
    ca-certificates \
    tini

ENTRYPOINT ["/sbin/tini", "--"]

ENV PATH $PATH:/root

ARG LNTOP_CONF_PATH

# copy the binaries and entrypoint from the builder image.
COPY --from=builder /go/bin/lntop /bin/

WORKDIR /root

COPY "home" .
