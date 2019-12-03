################################
# STEP 1 build executable binary
################################
FROM golang:alpine as builder

# Install git (required by godeps)
RUN apk update && apk add --no-cache git curl openssh-client gcc g++ musl-dev

ADD . /go/src/geth-lb

WORKDIR /go/src/geth-lb/server

# Install dep
RUN go get github.com/btcsuite/btcd/btcec
RUN go get -d -v .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o /go/bin/geth-lb
RUN chmod 755 /go/bin/geth-lb

############################
# STEP 2 build a small image
############################
FROM scratch

COPY --from=builder /go/bin/geth-lb /go/bin/geth-lb

CMD ["/go/bin/geth-lb"]