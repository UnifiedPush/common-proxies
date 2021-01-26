# golang-unified-push-rewrite-proxy

## Installing
1. Download the binaries onto your server
2. Use the following nginx config, or the equivalent for your reverse proxy.
```nginx 
location ~ ^/(FCM|UP)/ {    
        proxy_pass            http://127.0.0.1:5000;
}
```
The values FCM and UP will depend on which ones you actually have enabled. The :5000 will be the port you will specify as `-l` in the service file.
3. Place the systemd service file in /etc/systemd/system/up-rewrite-proxy.service . Make sure to edit the contents of the file and the flags you want to enable.

4. `systemctl enable --now up-rewrite-proxy`

5. You're done!


## Providers
### FCM

This is meant to be hosted by the app developers or someone who has access to the Firebase settings for that project. This is because the --fcm parameter is the key that's present in the Cloud Messaging section of your projects Firebase settings.

### Gotify

This is primarily meant to be hosted on the same machine as the Gotify server. Running it on a different machine hasn't been tested yet but you can share information about that in this repos issues.
