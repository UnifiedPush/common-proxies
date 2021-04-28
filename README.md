# Go UnifiedPush Rewrite Proxy

## Installing
1. Download the binary and [example config file](./config.toml) onto your server
2. Use the following nginx config, or the equivalent for your reverse proxy.
```nginx 
location ~ ^/(FCM|UP|_matrix)/ {    
        proxy_pass            http://127.0.0.1:5000;
}
```
The values FCM, UP, etc will depend on which ones you actually have enabled. The :5000 will be the port you specify to the listen directive in the config file.

3. Place the [systemd service file](./up-rewrite-proxy.service) in /etc/systemd/system/up-rewrite-proxy.service . Make sure to edit the contents of the file.

4. `systemctl enable --now up-rewrite-proxy`

5. You're done!


## Rewrite Proxy
### FCM

This is meant to be hosted by the app developers or someone who has access to the Firebase settings for that project. The FCM key under `rewrite.fcm` is this secret key.

### Gotify

This is primarily meant to be hosted on the same machine as the Gotify server. Running it on a different machine hasn't been tested yet but you can share information about that in this repo's issues.

## Gateway
Note: Gateways cannot connect to localhost or other private IPs. IPv6 support missing is a known bug.

### Matrix
Gateways matrix push until [MSC 2970](https://github.com/matrix-org/matrix-doc/pull/2970) is accepted.  
`["notification"]["devices"][0]["pushkey"]` is the UP endpoint this gateways to.

## Note
* Not all architectures in the releases have been tested.
