# Configuration

See [the example configuration file](../example-config.toml) for how a toml configuration should be arranged.

| Description                 | TOML Name              | Environment Variable Name | Type        | More Info                                                                     |
| :---:                       | ---                    | ---                       | ---         | ---                                                                           |
| HTTP Listener Address       | listenAddr             | UP_LISTEN                 | string      | This doesn't have any effect inside docker.                                   |
| Verbose logs                | verbose                | UP_VERBOSE                | boolean     | Detailed logs or not. It is recommended to always set this to true.           |
| Enable Matrix Gateway       | gateway.matrix.enable  | UP_GATEWAY_MATRIX_ENABLE  | boolean     |                                                                               |
| Enable Gotify Rewrite Proxy | rewrite.gotify.enable  | UP_REWRITE_GOTIFY_ENABLE  | boolean     |                                                                               |
| Gotify forwarding address   | rewrite.gotify.address | UP_REWRITE_GOTIFY_ADDRESS  | string      | What is the domain of your Gotify server. This has to be a `host:port` or `host` if you want the default port for the scheme. |
| Gotify forwarding scheme    | rewrite.gotify.scheme  | UP_REWRITE_GOTIFY_SCHEME  | string      | `http` or `https`                                                             |
| Enable FCM Rewrite Proxy    | rewrite.fcm.enable     | UP_REWRITE_FCM_ENABLE     | boolean     |                                                                               |
| Firebase Server Key for FCM | rewrite.fcm.key        | UP_REWRITE_FCM_KEY        | string      |                                                                               |
| Allowed Gateway Hosts       | gateway.AllowedHosts   | UP_GATEWAY_ALLOWEDHOSTS   | string list | See relevant section below                                                    |



## Gateway Allowed Hosts

Most people shouldn't use this.

This is used if you need to allow gatewaying to an internal or local host. Use this with caution and only allow as few hosts as required. All public addresses are allowed by default.  
This takes in `<host>` if the default port for HTTP and/or HTTPS is expected or `<host>:<port>` if not.
`<host>` can also be an IP address if such a request is expected (shouldn't be the case for **most** setups).  

The port only needs to be included if it's something other than 80 or 443, but if so, entries for both HTTP and HTTPS should be included.

Example:
```toml
AllowedHosts = ["abc.localhost:8443", "abc.localhost:8080", "myinternaldomain.local"] 
```

In environment variables, this is a comma seperated list:
```env
UP_GATEWAY_ALLOWEDHOSTS="abc.localhost:8443,abc.localhost:8080,myinternaldomain.local"
```

## Configuration file location

By default the configuration file should be located at `config.toml` in the current working directory (the one from which the command is run). This can be changed by adding the `-c` flag when running the application on the command line, and passing an alternate path to that.
