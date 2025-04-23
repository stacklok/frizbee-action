FROM index.docker.io/library/golang:1.23.7-alpine@sha256:e438c135c348bd7677fde18d1576c2f57f265d5dfa1a6b26fca975d4aa40b3bb

COPY . /home/src
WORKDIR /home/src
RUN go build -o /bin/action ./

ENTRYPOINT [ "/bin/action" ]
