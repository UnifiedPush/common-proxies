# Reverse-Proxy Configuration

## Common-Proxies

All of the following examples will have Common-Proxies hosted at `127.0.0.1:5000`. You can adjust the ports according to your needs.

### Nginx

```nginx
server {

	listen 0.0.0.0:80;
	listen [::1]:80 ;

	listen 0.0.0.0:443 ssl;
	listen [::1]:443 ssl;

	# Here goes your domain / subdomain
	server_name unifiedpushproxy.example.com;

	# this sends traffic to common-proxies
	location ~ ^/(wpfcm|UP|_matrix) {
		proxy_pass			http://127.0.0.1:5000;
	}
}
```

### Apache

```apache
ServerName unifiedpushproxy.example.org

# Send these 3 paths to common-proxies
ProxyPass "/_matrix" http://127.0.0.1:5000/_matrix
ProxyPass "/generic" http://127.0.0.1:5000/generic
ProxyPass "/wpfcm" http://127.0.0.1:5000/wpfcm
```

### Caddy

This snippet can be placed in a Caddyfile.
```caddy
unifiedpushproxy.example.org {
    @rewrite_proxy {
        path /generic* /_matrix* /wpfcm*
    }
    reverse_proxy @rewrite_proxy 127.0.0.1:5000
}
```

### Traefik v2

Check the example [docker-compose.yml](./docker-compose-traefik.yml) for Traefik which configures the reverse proxy using labels.

The same settings could also be written in a yaml or toml format for the Traefik File provider, which can be used when common proxies is not running as a docker service.

Here is a toml example:

```toml
[http.routers]
  [http.routers.commonproxies]
    entryPoints = ["websecure"]
    rule = "Host(`unifiedpushproxy.example.org`) && PathPrefix(`/generic`, `/wpfcm`, `/_matrix`)"
    service = "commonproxies-service"

    [http.routers.commonproxies.tls]
      certResolver = "myresolver"

[http.services]
  [http.services.commonproxies-service.loadBalancer]
    [[http.services.commonproxies-service.loadBalancer.servers]]
      url = "http://127.0.0.1:5000/"
```
