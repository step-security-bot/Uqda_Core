FROM docker.io/golang:alpine@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 as builder

COPY . /src
WORKDIR /src

ENV CGO_ENABLED=0

RUN apk add git && ./build && go build -o /src/genkeys cmd/genkeys/main.go

FROM docker.io/alpine

COPY --from=builder /src/uqda /usr/bin/uqda
COPY --from=builder /src/uqdactl /usr/bin/uqdactl
COPY --from=builder /src/genkeys /usr/bin/genkeys
COPY contrib/docker/entrypoint.sh /usr/bin/entrypoint.sh

# RUN addgroup -g 1000 -S uqda \
#  && adduser -u 1000 -S -g 1000 --home /etc/uqda uqda
#
# USER uqda
# TODO: Make running unprivileged work

VOLUME [ "/etc/uqda" ]

ENTRYPOINT [ "/usr/bin/entrypoint.sh" ]
