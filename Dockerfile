FROM golang:1.15-buster as build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY ./ ./
RUN go build -o up-rewrite

FROM debian:stable-slim
ENV UP_PROXY_SERVER_PORT="80"
WORKDIR /app
RUN export DEBIAN_FRONTEND=noninteractive && apt-get update && apt-get install -yq \
  curl \
  && rm -rf /var/lib/apt/lists/*
ADD config.toml /app/
COPY --from=build /src/up-rewrite /app/
#HEALTHCHECK --interval=30s --timeout=5s --start-period=5s CMD curl --fail http://localhost:$UP_PROXY_SERVER_PORT/UP || exit 1
#TODO

RUN chown www-data -R .
USER www-data
EXPOSE 5000
ENTRYPOINT ["./up-rewrite"]
