FROM index.docker.io/library/golang:1.25.0-alpine@sha256:f18a072054848d87a8077455f0ac8a25886f2397f88bfdd222d6fafbb5bba440

COPY . /home/src
WORKDIR /home/src
RUN go build -o /bin/action ./

ENTRYPOINT [ "/bin/action" ]
