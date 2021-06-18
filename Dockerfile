FROM golang:latest

RUN mkdir /code
WORKDIR /code

COPY . .

ENV GIN_MODE release

RUN go build

CMD ["./srt-relay"]
