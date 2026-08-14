# Document Harbor registry polling and webhooks

## Summary
Add an end-to-end Harbor guide covering Registry V2 project paths, semantic-version and mutable-tag polling, private project credentials, native `PUSH_ARTIFACT` webhook setup, and TLS considerations. Correct the native Harbor webhook target from the generic registry endpoint to `/v1/webhooks/harbor`.

## Testing
`npm ci` completed with an isolated cache, and `NODE_OPTIONS=--openssl-legacy-provider npm run build` generated the VuePress site successfully.
