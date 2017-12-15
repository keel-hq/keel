Get a list of the currently active webhooks for each room or for a specific room

List all rooms:

  $ go run main.go -token="$TOKEN"

List a specific room:

  $ go run main.go -token="$TOKEN" -room="$ROOM"

Delete a webhook:

  $ go run main.go -token="$TOKEN" -room="$ROOM" -action="delete" -webhook="$WEBHOOK"

Create a webhook:

  $ go run main.go -token="$TOKEN" -room="$ROOM" -action="delete" \
    -name="$NAME" \
    -event="$EVENT" \
    -pattern="$PATTERN" \
    -url="$URL"
