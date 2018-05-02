FROM golang:1.10.1-stretch

WORKDIR /go/src/ubot
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...
RUN ./build-plugins.sh

CMD ["ubot"]