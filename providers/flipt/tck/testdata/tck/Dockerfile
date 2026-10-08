# The testbed image is the flipt server plus the TCK control server. The
# control server runs as PID 1 and starts, stops and reseeds the /flipt process
# inside the same container, so the OpenFeature provider TCK can simulate
# backend outages and flag changes without ever restarting the container.
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd ./cmd
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -mod=mod -ldflags="-s -w" -o /out/control ./cmd/control

FROM ghcr.io/flipt-io/flipt:v2
COPY --from=build /out/control /control
EXPOSE 8080 8888
USER flipt
ENTRYPOINT ["/control"]
CMD []
