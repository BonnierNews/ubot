FROM golang:1.10.1-stretch

WORKDIR $GOPATH/src/github.com/BonnierNews/ubot
ADD https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure --vendor-only
COPY . .
RUN ./build-plugins.sh && go build .

CMD ["./ubot"]