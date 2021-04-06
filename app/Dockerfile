FROM golang:1.15.1-alpine3.12 AS build-env
WORKDIR /tmp/app
RUN CGO_ENABLED=0 GOOS=linux go build 

FROM scratch
COPY --from=build-env /tmp/app /bin/app
EXPOSE 8080
CMD ["/bin/app -listen-address=:8080"]