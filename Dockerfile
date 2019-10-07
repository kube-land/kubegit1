ARG IMAGE=alpine:3.9.3

FROM golang:1.12.3-alpine as builder
WORKDIR ${GOPATH}/src/github.com/appspero/kube-git
COPY . ./
RUN apk add --update gcc libc-dev linux-headers
RUN CGO_ENABLED=0 GOOS=linux go build -o /usr/bin/kube-git cmd/kubegit/main.go

FROM ${IMAGE}
RUN apk update && apk add git ca-certificates && rm -rf /var/cache/apk/*
RUN update-ca-certificates
COPY --from=builder /usr/bin/kube-git /usr/bin/
ENTRYPOINT ["/usr/bin/kube-git"]
