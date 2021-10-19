# Docker-Compose Quick start

If you know how to use docker-compose, the following is the quickest way to set up the whole common-proxies, gotify, caddy stack.

1. Download this [docker-compose.yml](./docker-compose-quick.yml) in a new directory.

1. Save one of the following files to .env in the same directory.

    If HTTPS is needed and the ports 443 and 80 are unused:
    Change the domain to yours.

    ```env
    DOMAIN=mydomain.example.com

    LISTEN_DOMAIN="http://${DOMAIN} https://${DOMAIN}"
    HTTP=80
    HTTPS=443
    ```

    If you have another reverse proxy doing TLS and have that running on ports 80 and 443:
    Point them to localhost:5135 or change the port to one of your choice.

    ```env
    HTTP=127.0.0.1:5135

    DOMAIN=*
    LISTEN_DOMAIN="http://${DOMAIN}"
    HTTPS=127.0.0.1:0 # essentially disables it
    ```

    These two are just basic configurations, things can be modified for more custom needs.

1. Run `docker-compose up -d` in that directory.

The linked docker compose file can be modified to suite your needs. Other environment variable configuration options are available in [config.md](config.md).
