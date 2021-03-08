FROM golang:1.16 as builder

RUN mkdir /app
WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -v -o esp-echo

FROM alpine:3.13

COPY --from=builder /app/esp-echo /esp-echo

CMD ["/esp-echo"]
