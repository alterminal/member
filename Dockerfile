FROM golang:1.22.5 AS build-stage
WORKDIR /
COPY . .
RUN go build -o /main

FROM ubuntu:latest AS production-stage
RUN apt-get update && apt-get install tzdata

ENV TZ=Asia/Shanghai

COPY --from=build-stage /main /main
EXPOSE 8080
CMD ["/main"]