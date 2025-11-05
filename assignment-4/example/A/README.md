# Device A

Onboard the device.
```bash
go run ../../cmd/client/main.go device onboard
```

Push some files to be shared.
```bash
go run ../../cmd/client/main.go share push --files "files/**/*.md"
```

You will receive a secret and an id, send it to device B.

Then proceed with device B [instructions](../B/README.md).

## Usefull commands

At any time, you can inspect the status of a share.
```bash
go run ../../cmd/client/main.go share summary --id <id> --share-secret <secret>
```