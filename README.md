# UnifiedPush Common-Proxies

Common-Proxies is a set of rewrite proxies and gateways for UnifiedPush. See the following diagram for more info regarding where rewrite proxies and gateways fit in UnifiedPush.
![UnifiedPush service connection diagram](https://unifiedpush.org/img/diagram.png)

## Installing

1. Download the binary and [example config file](./example-config.toml) onto your server
. Place the [systemd service file](./up-rewrite-proxy.service) in /etc/systemd/system/up-rewrite-proxy.service . Make sure to edit the contents of the file.
1. `systemctl enable --now up-rewrite-proxy`
1. Install the [reverse-proxy](#reverse-proxy)

### Docker-Compose with Gotify quick start

1. Download this [docker-compose.yml](./docker-compose.yml) in a new directory.

1. Save one of the following files to .env in the same directory, depending on your needs.

    If HTTPS is needed and the ports 443 and 80 have nothing else running on them.

    ```env
    DOMAIN=mydomain.example.com
    UP_VERSION=v1.1

    LISTEN_DOMAIN="http://${DOMAIN} https://${DOMAIN}"
    HTTP=80
    HTTPS=443
    ```

    If you have another reverse proxy doing TLS and have that running on ports 80 and 443.

    ```env
    HTTP=127.0.0.1:4567
    UP_VERSION=v1.1

    DOMAIN=*
    LISTEN_DOMAIN="http://${DOMAIN} https://${DOMAIN}"
    HTTPS=127.0.0.1:0 # essentially disables it
    ```

    These two are just basic configurations, things can be modified for more custom needs.

1. Run `docker-compose up -d` in that directory.

The linked docker compose file can be modified to suite your needs. Other configuration options as environment variables are available in the [example config file](./example-config.toml).

### Reverse Proxy

Use the following nginx config, or the equivalent for your reverse proxy. The following snippet goes inside the same domain as your push provider (probably Gotify).

```nginx
location ~ ^/(FCM|UP|_matrix) {    
        proxy_pass            http://127.0.0.1:5000;
}
```

The 127.0.0.1 will be the host, which when installed directly is probably 127.0.0.1, but when installed using docker is the container IP or the container hostname. The :5000 will be the port you specify to the listen directive in the config file or the docker port flag.

## Rewrite Proxy

Common-Proxies handles paths like /UP (Gotify) or /FCM (Firebase). Only traffic for these paths should be forwarded to common-proxies, where it can then convert the push message to the push-provider specific format.

### FCM

This is meant to be hosted by the app developers or someone who has access to the Firebase settings for that project. The FCM key under `rewrite.fcm` in the config file is this secret key.

### Gotify

This is primarily meant to be hosted on the same machine as the Gotify server. Running it on a different machine hasn't been tested yet but you can share information about that in this repo's issues.

## Gateway

A Gateway is meant to take push messages from an existing service (like Matrix) and convert it to the UnifiedPush format. While Gateways are primarily meant to be hosted by the App Developer, some Gateways (like the Matrix one) support discovery on the push provider domain to find self-hosted gateways. It's always optional to host gateways as the app developer usually should have one.

### Matrix

Gateways Matrix push notifications.  
`["notification"]["devices"][0]["pushkey"]` is the UP endpoint this gateways to.

## Note

* Not all architectures in the releases have been tested.
