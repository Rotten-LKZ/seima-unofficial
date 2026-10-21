# Cloudflare Workers OpenAPI 3.1

This is a Cloudflare Worker with OpenAPI 3.1 using [chanfana](https://github.com/cloudflare/chanfana) and [Hono](https://github.com/honojs/hono).

This is an example project made to be used as a quick start into building OpenAPI compliant Workers that generates the
`openapi.json` schema automatically from code and validates the incoming request to the defined parameters or request body.

## Get started

1. Sign up for [Cloudflare Workers](https://workers.dev). The free tier is more than enough for most use cases.
2. Clone this project and install dependencies with `npm install`
3. Run `wrangler login` to login to your Cloudflare account in wrangler
4. Run `wrangler deploy` to publish the API to Cloudflare Workers

## Project structure

1. Your main router is defined in `src/index.ts`.
2. Each endpoint has its own file in `src/endpoints/`.
3. For more information read the [chanfana documentation](https://chanfana.pages.dev/) and [Hono documentation](https://hono.dev/docs).

## Development

1. Run `wrangler dev` to start a local instance of the API.
2. Open `http://localhost:8787/` in your browser to see the Swagger interface where you can try the endpoints.
3. Changes made in the `src/` folder will automatically trigger the server to reload, you only need to refresh the Swagger interface.

## Configuration

### D1 database

`wrangler.jsonc` binds a D1 database named `streaming_logs` as `env.DB`. The `database_id` field there is **not sensitive** — it is a resource identifier, not a credential. It is safe to commit; access is still gated by your Cloudflare account/API token.

After creating the database, copy the printed id into `wrangler.jsonc` and apply migrations:

```sh
npx wrangler d1 create streaming_logs
# edit wrangler.jsonc -> "database_id": "<id from output>"
npx wrangler d1 migrations apply streaming_logs --remote
```

### Environment variables & secrets

| Name | Type | Where set | Purpose |
|---|---|---|---|
| `ADMIN_TOKEN` | **secret** | `npx wrangler secret put ADMIN_TOKEN` (production); `.dev.vars` (local dev) | Bearer token required for all `/admin/*` endpoints |

Setup:

```sh
# local dev: copy the template and fill in a value
cp .dev.vars.example .dev.vars

# production: store as an encrypted secret (never put it in wrangler.jsonc)
npx wrangler secret put ADMIN_TOKEN
```

`.dev.vars` is git-ignored; `.dev.vars.example` is the committed template. Never put the real token in `wrangler.jsonc` `vars` — that ships in plaintext with your Worker bundle.
