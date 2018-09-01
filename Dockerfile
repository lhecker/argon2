FROM golang:alpine AS build

WORKDIR /go/src/github.com/lhecker/argon2
COPY . .

RUN apk add --no-cache \
    g++

RUN mkdir -p build \
    && go build -o ./build/example ./examples


FROM alpine

COPY --from=build /go/src/github.com/lhecker/argon2/build/example .
ENTRYPOINT [ "/example" ]
