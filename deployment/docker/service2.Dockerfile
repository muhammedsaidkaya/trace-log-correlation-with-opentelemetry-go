FROM golang:1.19 as builder
RUN mkdir /build
WORKDIR /build
COPY . .
RUN go mod download


RUN CGO_ENABLED=0 GOOS=linux go build -a -o service2 .

FROM alpine:3
COPY --from=builder /build/service2 .
EXPOSE 8080
# Executable
ENTRYPOINT [ "./service2" ]