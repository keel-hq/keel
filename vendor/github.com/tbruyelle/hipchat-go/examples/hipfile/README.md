hipfile
=====

Sends the given file to the specified room or user.

##### Usage

```bash
go build
./hipfile --token=<your auth token> --room=<room id> --path=<file path>
```

##### Example

Give it a try with the gopher.png file

```bash
go build
./hipfile --token=<your auth token> --room=<room id> --path=gopher.png --message="Check out this one!"
```

