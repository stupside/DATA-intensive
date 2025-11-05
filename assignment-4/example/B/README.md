# Device B

Onboard the device.
```bash
go run ../../cmd/client/main.go device onboard
```

Take the id and the secret device A sent you and pull the files.
```bash
go run ../../cmd/client/main.go share pull --id <id> --share-secret <secret> --output-dir ./out
```

## Usefull commands

At any time, you can inspect the status of a share.
```bash
go run ../../cmd/client/main.go share summary --id <id> --share-secret <secret>
```