FROM golang as builder
WORKDIR /loki-docker-sd
COPY . .
RUN CGO_ENABLED=0 go build -ldflags='-s -w -extldflags="-static"' .

FROM alpine
COPY --from=builder /loki-docker-sd/loki-docker-sd /usr/bin/loki-docker-sd
WORKDIR /
VOLUME ["/targets"]
ENTRYPOINT ["loki-docker-sd", "-f=/targets/targets.json"]
