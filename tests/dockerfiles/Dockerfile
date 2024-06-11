FROM index.docker.io/library/golang:alpine3.19@sha256:65b5d2d0a312fd9ef65551ad7f9cb5db1f209b7517ef6d5625cfd29248bc6c85
RUN apk add git 

COPY . /home/src
WORKDIR /home/src
RUN go build -o /bin/action ./

ENTRYPOINT [ "/bin/action" ]
