# @machanirobotics/loom-network svelte example

This Svelte app demonstrates `@machanirobotics/loom-network` in a tabbed UI:

- HTTP example
- GraphQL example
- WebSocket example

Each tab runs a real request and shows:

- request payload
- normalized transport meta
- response/error payload

## Run

From this folder:

```sh
bun install
bun run dev
```

Open the local URL printed by Vite and use the tabs on the page.

## Notes

- The app imports runtime API from `@machanirobotics/loom-network`.
- The package is consumed locally through `link:@machanirobotics/loom-network` in `package.json`.
