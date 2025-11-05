# Example

This folder hold an example of interaction between two devices `A` and `B` where `A` wants to send files to `B`.

Follow the instructions [for A](A/README.md) and [for B](B/README.md).

## Usefull commands

Get a list of all the shares created by the devices.
```bash
go run ../../cmd/client/main.go device shares
```

Get a list of past device connections.
```bash
go run ../../cmd/client/main.go device connections
```