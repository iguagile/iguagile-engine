FROM golang:alpine AS build

WORKDIR $GOPATH/src/github.com/iguagile/iguagile-engine

COPY . .

RUN CGO_ENABLED=0 go build -o /app cmd/relay/main.go

FROM alpine
RUN apk add --no-cache tzdata ca-certificates
COPY --from=build /app /app

CMD ["/app"]
