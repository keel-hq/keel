# Clarify Harbor registry support and mutable-tag updates

## Summary
Document that Keel supports Harbor through Docker Registry v2 polling and native Harbor webhooks. Explain that mutable tags such as `latest` require the `force` policy with tag matching, add regression coverage for Harbor project paths and digest polling, and preserve Harbor webhook digests in submitted events.

## Testing
`go test -count=1 ./...` passed.
