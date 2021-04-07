FROM golang:1.15.1-alpine3.12 AS build-env
WORKDIR /tmp/workdir

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build
RUN cd pinger && CGO_ENABLED=0 GOOS=linux go build

FROM scratch
COPY --from=build-env /tmp/workdir/app /bin/app
COPY --from=build-env /tmp/workdir/pinger/pinger /bin/pinger

CMD ["/bin/app"]