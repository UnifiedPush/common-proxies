# UnifiedPush Common-Proxies

Common-Proxies is a set of rewrite proxies and push gateways for UnifiedPush. See the following diagram for more info regarding where rewrite proxies and push gateways fit in UnifiedPush.
![UnifiedPush service connection diagram](https://unifiedpush.org/img/diagram.png)

Common-Proxies is most commonly used to do the following:
  1. Self-Hosting the [Matrix](//matrix.org) to UnifiedPush push gateway.
  1. App Developers only: Running embedded FCM distributor.

## Installing

To install this the typical way, go to [install.md](docs/install.md).  
If you know docker-compose and want to quickly set it up, go to [docker-quickstart.md](docs/docker-quickstart.md).

Documentation for configuration can be found at [config.md](docs/config.md).

For more details about how this works, read on -

## Rewrite Proxy

Common-Proxies handles paths like /FCM (Firebase). Only traffic for these paths should be forwarded to common-proxies, where it can then convert the push message to the push-provider specific format.

### FCM

This is meant to be hosted by the app developers or someone who has access to the Firebase settings for that project. The FCM key under `rewrite.fcm` in the config file is this secret key.

## Gateway

A Gateway is meant to take push messages from an existing service (like Matrix) and convert it to the UnifiedPush format. While Gateways are primarily meant to be hosted by the App Developer, some Gateways (like Matrix) support discovery on the push provider domain to find self-hosted gateways. It's always optional to host gateways as the app developer must have one.

### Matrix

Gateways Matrix push notifications.  
`["notification"]["devices"][0]["pushkey"]` is the UP endpoint this gateways to.

### Generic

Appends WebPush AESGCM headers to the message body and passes on the message.

## Note

* Not all architectures in the releases have been tested.
