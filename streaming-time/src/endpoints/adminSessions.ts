import { OpenAPIRoute } from "chanfana";
import { z } from "zod";
import { type AppContext, Session } from "../types";

export class SessionList extends OpenAPIRoute {
	schema = {
		tags: ["Admin"],
		summary: "List sessions, optionally filtered by streamer",
		request: {
			query: z.object({
				streamer_id: z.coerce.number().int().optional(),
			}),
		},
		responses: {
			"200": {
				description: "Sessions (times in unix ms, UTC)",
				content: { "application/json": { schema: z.object({ sessions: Session.array() }) } },
			},
		},
	};

	async handle(c: AppContext) {
		const data = await this.getValidatedData<typeof this.schema>();
		const { streamer_id } = data.query;
		const stmt =
			streamer_id !== undefined
				? c.env.DB.prepare(
						"SELECT id, streamer_id, start_time, end_time FROM sessions WHERE streamer_id = ? ORDER BY start_time",
					).bind(streamer_id)
				: c.env.DB.prepare(
						"SELECT id, streamer_id, start_time, end_time FROM sessions ORDER BY start_time",
					);
		const { results } = await stmt.all();
		return { sessions: results ?? [] };
	}
}

export class SessionCreate extends OpenAPIRoute {
	schema = {
		tags: ["Admin"],
		summary: "Create session (times in unix ms, UTC)",
		request: {
			body: {
				content: {
					"application/json": {
						schema: z.object({
							streamer_id: z.number().int(),
							start_time: z.number().int(),
							end_time: z.number().int().nullable().optional(),
						}),
					},
				},
			},
		},
		responses: { "200": { description: "Created session" } },
	};

	async handle(c: AppContext) {
		const data = await this.getValidatedData<typeof this.schema>();
		const { streamer_id, start_time, end_time } = data.body;
		const row = await c.env.DB.prepare(
			"INSERT INTO sessions (streamer_id, start_time, end_time) VALUES (?, ?, ?) RETURNING id, streamer_id, start_time, end_time",
		)
			.bind(streamer_id, start_time, end_time ?? null)
			.first();
		return row;
	}
}

export class SessionUpdate extends OpenAPIRoute {
	schema = {
		tags: ["Admin"],
		summary: "Update session (times in unix ms, UTC)",
		request: {
			params: z.object({ id: z.coerce.number().int() }),
			body: {
				content: {
					"application/json": {
						schema: z.object({
							streamer_id: z.number().int().optional(),
							start_time: z.number().int().optional(),
							end_time: z.number().int().nullable().optional(),
						}),
					},
				},
			},
		},
		responses: { "200": { description: "Updated session" } },
	};

	async handle(c: AppContext) {
		const data = await this.getValidatedData<typeof this.schema>();
		const { id } = data.params;
		const { streamer_id, start_time, end_time } = data.body;

		const fields: string[] = [];
		const values: unknown[] = [];
		if (streamer_id !== undefined) { fields.push("streamer_id = ?"); values.push(streamer_id); }
		if (start_time !== undefined) { fields.push("start_time = ?"); values.push(start_time); }
		if (end_time !== undefined) { fields.push("end_time = ?"); values.push(end_time); }
		if (fields.length === 0) return c.json({ error: "nothing to update" }, 400);

		values.push(id);
		const row = await c.env.DB.prepare(
			`UPDATE sessions SET ${fields.join(", ")} WHERE id = ? RETURNING id, streamer_id, start_time, end_time`,
		)
			.bind(...values)
			.first();
		if (!row) return c.json({ error: "not found" }, 404);
		return row;
	}
}

export class SessionDelete extends OpenAPIRoute {
	schema = {
		tags: ["Admin"],
		summary: "Delete session",
		request: { params: z.object({ id: z.coerce.number().int() }) },
		responses: { "200": { description: "Deleted" } },
	};

	async handle(c: AppContext) {
		const data = await this.getValidatedData<typeof this.schema>();
		const { id } = data.params;
		const res = await c.env.DB.prepare("DELETE FROM sessions WHERE id = ?").bind(id).run();
		if (res.meta.changes === 0) return c.json({ error: "not found" }, 404);
		return { deleted: true };
	}
}
