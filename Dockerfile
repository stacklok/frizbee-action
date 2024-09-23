FROM golang:1.23.1-alpine3.20@sha256:ac67716dd016429be8d4c2c53a248d7bcdf06d34127d3dc451bda6aa5a87bc06

COPY . /home/src
WORKDIR /home/src
RUN go build -o /bin/action ./

ENTRYPOINT [ "/bin/action" ]
