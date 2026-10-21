import { createMiddleware } from "hono/factory";

export const adminAuth = createMiddleware<{ Bindings: Env }>(async (c, next) => {
	const header = c.req.header("Authorization") ?? "";
	const token = header.startsWith("Bearer ") ? header.slice(7) : c.req.query("token");
	if (!c.env.ADMIN_TOKEN || token !== c.env.ADMIN_TOKEN) {
		return c.json({ error: "unauthorized" }, 401);
	}
	await next();
});
