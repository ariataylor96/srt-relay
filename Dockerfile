FROM golang:latest as build

RUN mkdir /srt-relay
WORKDIR /srt-relay

COPY main.go go.mod go.sum .

ENV GIN_MODE release

RUN go build
RUN ls

FROM ubuntu:latest

COPY --from=build /srt-relay .

CMD ["./srt-relay"]
