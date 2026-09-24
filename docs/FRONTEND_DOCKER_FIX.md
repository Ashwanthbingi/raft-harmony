# Frontend Docker Fix

## Root cause

The project uses TanStack Start through `@lovable.dev/vite-tanstack-config`. That wrapper enables Nitro for production builds and defaults Nitro to the `cloudflare-module` preset. The resulting `.output/server/index.mjs` is a Worker-style module: importing it defines the Nitro application and exports its `fetch` handler, but it does not create a Node HTTP listener. Running it with `node` therefore exits successfully with code 0.

The Docker image also copied `/app/.output` into the runtime work directory as `/app/server` and `/app/public`, while its command referenced `/app/.output/server/index.mjs`. That layout did not match the command.

## Correct production configuration

`vite.config.ts` now explicitly sets:

```ts
nitro: {
  preset: "node-server",
}
```

This is compatible with the installed Nitro `3.0.260603-beta` version and produces a Node-compatible Nitro server entry point. TanStack Start remains enabled, including the custom `src/server.ts` SSR wrapper and server functions.

## Docker runtime configuration

`Dockerfile.frontend` preserves the production runtime settings:

- `HOST=0.0.0.0`
- `PORT=3000`
- `EXPOSE 3000`
- `dumb-init` as PID 1
- `node .output/server/index.mjs` as the real server process

The generated output is copied to `/app/.output`, matching the command. Compose continues to map `8080:3000`, and `VITE_RAFT_NODES` remains configured in `docker-compose.yml`.

## Verification

Build and start the stack:

```sh
docker compose build frontend --no-cache
docker compose up -d
docker compose ps
docker compose logs frontend --tail=100
```

Verify the server inside and outside the container:

```sh
docker exec raft-frontend wget -qO- http://localhost:3000/
curl http://localhost:8080/
docker compose ps
```

The frontend should remain `Up` and serve the dashboard HTML. In this checkout, the existing Raft image health check requests `/health`, but the backend closes that request without an HTTP response, so Compose reports the nodes as `unhealthy` even though their processes remain running. That pre-existing backend health-check mismatch is separate from the frontend restart fix.
