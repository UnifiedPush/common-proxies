# Configuration

See [the example configuration file](../example-config.toml) for how a toml configuration should be arranged.

| Description                       | TOML Name                    | Environment Variable Name       | Type                 | More Info                                                                                                                                                                  |
| :---:                             | ---                          | ---                             | ---                  | ---                                                                                                                                                                        |
| HTTP Listener Address             | listenAddr                   | UP_LISTEN                       | string               | This doesn't have any effect inside docker.                                                                                                                                          |
| Verbose logs                      | verbose                      | UP_VERBOSE                      | boolean              | Detailed logs or not. It is recommended to always set this to true.                                                                                                                  |
| Gateway User Agent                | UserAgentID                  | UP_UAID                         | string               | A user agent comment for gateway forwarded requests. Useful for debugging (and rate limits for big gateways). Example: "matrix.gateway.unifiedpush.org by unifiedpush.org"           |
| Enable Matrix Gateway             | gateway.matrix.enable        | UP_GATEWAY_MATRIX_ENABLE        | boolean              |                                                                                                                                                                                      |
| Enable FCM Rewrite Proxy          | rewrite.webpushfcm.enable    | UP_REWRITE_WEBPUSH_FCM_ENABLE   | boolean              |                                                                                                                                                                                      |
| VAPID private key for FCM         | rewrite.fcm.credentialsPath  | UP_REWRITE_WEBPUSH_FCM_CREDENTIALS_PATH | string       | WebPush requests to FCM needs a VAPID authorization. The private key used to generate the authorization is loaded from this path. To generate a new one, run `common-proxies -vapid` |
| Allowed Gateway Hosts             | gateway.AllowedHosts         | UP_GATEWAY_ALLOWEDHOSTS         | string list          | See relevant section below                                                                                                                                                           |
| Enable AESGCM Gateway             | gateway.aesgcm.enable        | UP_GATEWAY_AESGCM_ENABLE        | boolean              | Enable the AESGCM gateway on /aesgcm to convert old webpush requests to UnifiedPush compatible ones   |

__Deprecated configurations__

| Description                       | TOML Name                    | Environment Variable Name       | Type                 | More Info                                                                                                                                                                  |
| :---:                             | ---                          | ---                             | ---                  | ---                                                                                                                                                                        |
| Enable FCM Rewrite Proxy | rewrite.fcm.enable           | UP_REWRITE_FCM_ENABLE           | boolean              |                                                                                                                                                                            |
| Firebase Credentials for FCM | rewrite.fcm.credentialsPath  | UP_REWRITE_FCM_CREDENTIALS_PATH | string               | An FCM request to any hostname will be forwarded with credentials loaded from this path. Not recommended, use per hostname credentials if possible.                        |
| Firebase Credentials per hostname | rewrite.fcm.CredentialsPaths | none                            | map[hostname] = path | Specify the hostname that will be receiving requests and the credentials path that request should be forwarded with.                                                       |

## Gateway Allowed Hosts

Most people shouldn't use this. Only look at this if you're running into errors with a normal setup.

As long as the DNS on the common-proxies host returns a public IP for your gateway target (your push provider), you don't need this.
This is used if you need to allow gatewaying to an internal or local host. Use this with caution and only allow as few hosts as required. All public addresses are allowed by default.  
This takes in `<host>` if the default port for HTTP and/or HTTPS is expected or `<host>:<port>` if not.
`<host>` can also be an IP address if such a request is expected (shouldn't be the case for **most** setups).  

The port only needs to be included if it's something other than 80 or 443, but if so, entries for both HTTP and HTTPS should be included.

Example:
```toml
AllowedHosts = ["abc.localhost:8443", "abc.localhost:8080", "myinternaldomain.local"] 
```

In environment variables, this is a comma separated list:
```env
UP_GATEWAY_ALLOWEDHOSTS="abc.localhost:8443,abc.localhost:8080,myinternaldomain.local"
```

## Configuration file location

By default the configuration file should be located at `config.toml` in the current working directory (the one from which the command is run). This can be changed by adding the `-c` flag when running the application on the command line, and passing an alternate path to that.
