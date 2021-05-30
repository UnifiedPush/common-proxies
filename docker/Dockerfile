FROM amd64/debian:stable-slim
ENV UP_PROXY_SERVER_PORT="80"
WORKDIR /app
RUN export DEBIAN_FRONTEND=noninteractive && apt-get update && apt-get install -yq \
  tzdata \
  curl \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/*
ADD up-rewrite config.toml /app/
#HEALTHCHECK --interval=30s --timeout=5s --start-period=5s CMD curl --fail http://localhost:$UP_PROXY_SERVER_PORT/UP || exit 1
#TODO
EXPOSE 80
ENTRYPOINT ["./up-rewrite"]
