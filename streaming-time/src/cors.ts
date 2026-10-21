import { createMiddleware } from "hono/factory";

const ALLOWED_SUFFIX = ".loli.cn.com";
const ALLOWED_EXACT = "loli.cn.com";

export const corsLoli = createMiddleware<{ Bindings: Env }>(async (c, next) => {
	const origin = c.req.header("Origin");
	let allowOrigin: string | undefined;

	if (origin) {
		try {
			const host = new URL(origin).hostname;
			if (host === ALLOWED_EXACT || host.endsWith(ALLOWED_SUFFIX)) {
				allowOrigin = origin;
			}
		} catch {
			// invalid origin header -> no CORS allowance
		}
	}

	if (c.req.method === "OPTIONS") {
		if (!allowOrigin) return c.body(null, 403);
		return new Response(null, {
			status: 204,
			headers: {
				"Access-Control-Allow-Origin": allowOrigin,
				"Access-Control-Allow-Methods": "GET,POST,PUT,DELETE,OPTIONS",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
				"Access-Control-Max-Age": "86400",
				Vary: "Origin",
			},
		});
	}

	await next();

	if (allowOrigin) {
		c.res.headers.set("Access-Control-Allow-Origin", allowOrigin);
		c.res.headers.append("Vary", "Origin");
	}
});
