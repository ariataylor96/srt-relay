FROM golang:latest as build

RUN mkdir /srt-relay
WORKDIR /srt-relay

COPY *.go ws-handler.go go.mod go.sum .

RUN go build

FROM ubuntu:latest

COPY --from=build /srt-relay .

ENV GIN_MODE release

CMD ["./srt-relay"]
