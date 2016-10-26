FROM golang:1.7-alpine

# copy the source files
COPY . /go/src/github.com/san-lab/gastracker

# install dependencies and build executable
RUN apk add --no-cache git build-base && \
    go install github.com/san-lab/gastracker && \
    apk del git build-base

# set the command
CMD /go/bin/gastracker
