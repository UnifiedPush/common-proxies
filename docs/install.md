# Installation

After following one of the two sections below, set up a [reverse proxy](reverse_proxy.md).

## Linux

1. Download the [latest binary](https://github.com/UnifiedPush/common-proxies/releases) onto your server.
1. Create a new `config.toml` file based on the [example config file](./example-config.toml). See <config.md> for reference.

### Service management

The following instructions are for SystemD, the service manager that most people use by default. If you don't use it, replace this step with your prefered one.

Place the following into /etc/systemd/system/up-rewrite-proxy.service. Make sure to replace `{{install_directory}}` with whichever directory the binary is in; and replace `{{config_directory}}` with whichever folder you config.toml is in. (it is possible to make both the same directory if you want)

```systemd
[Unit]
Description=Rewrite Proxy for some UnifiedPush providers in Go
After=network.target
Requires=network.target

[Service]
Type=simple
WorkingDirectory={{config_directory}}
ExecStart={{install_directory}}/up-rewrite-proxy
Restart=always
RestartSec=10


[Install]
WantedBy=multi-user.target
```

Then, enable the service with
`systemctl enable --now up-rewrite-proxy`

### Logging

To see logs, run the following command

```sh
journalctl -xeu up-rewrite-proxy
```


## Docker

TODO
