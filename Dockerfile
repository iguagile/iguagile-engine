FROM golang:alpine AS build
ENV GO111MODULE=on

RUN apk add --no-cache git
RUN \
  cd $GOPATH/src/ && \
  mkdir -p github.com/iguagile && \
  cd github.com/iguagile && \
  git clone https://github.com/iguagile/iguagile-engine.git && \
  cd ./iguagile-engine && \
  GOOS=linux CGO_ENABLED=0 go build -a -o out cmd/sample-1/main.go && \
  cp out /app

FROM alpine
RUN apk add --no-cache tzdata ca-certificates
COPY --from=build /app /app

CMD ["/app"]
