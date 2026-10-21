import { OpenAPIRoute } from "chanfana";
import { z } from "zod";
import { type AppContext, Streamer } from "../types";

export class StreamerList extends OpenAPIRoute {
	schema = {
		tags: ["Admin"],
		summary: "List streamers",
		responses: {
			"200": {
				description: "All streamers",
				content: { "application/json": { schema: z.object({ streamers: Streamer.array() }) } },
			},
		},
	};

	async handle(c: AppContext) {
		const { results } = await c.env.DB.prepare(
			"SELECT id, name, room_id, uuid FROM streamers ORDER BY id",
		).all();
		return { streamers: results ?? [] };
	}
}

export class StreamerCreate extends OpenAPIRoute {
	schema = {
		tags: ["Admin"],
		summary: "Create streamer",
		request: {
			body: {
				content: {
					"application/json": {
						schema: z.object({
							name: z.string(),
							room_id: z.string(),
							uuid: z.string(),
						}),
					},
				},
			},
		},
		responses: {
			"200": {
				description: "Created streamer",
				content: { "application/json": { schema: Streamer } },
			},
		},
	};

	async handle(c: AppContext) {
		const data = await this.getValidatedData<typeof this.schema>();
		const { name, room_id, uuid } = data.body;
		const row = await c.env.DB.prepare(
			"INSERT INTO streamers (name, room_id, uuid) VALUES (?, ?, ?) RETURNING id, name, room_id, uuid",
		)
			.bind(name, room_id, uuid)
			.first();
		return row;
	}
}

export class StreamerUpdate extends OpenAPIRoute {
	schema = {
		tags: ["Admin"],
		summary: "Update streamer",
		request: {
			params: z.object({ id: z.coerce.number().int() }),
			body: {
				content: {
					"application/json": {
						schema: z.object({
							name: z.string().optional(),
							room_id: z.string().optional(),
							uuid: z.string().optional(),
						}),
					},
				},
			},
		},
		responses: { "200": { description: "Updated streamer" } },
	};

	async handle(c: AppContext) {
		const data = await this.getValidatedData<typeof this.schema>();
		const { id } = data.params;
		const { name, room_id, uuid } = data.body;

		const fields: string[] = [];
		const values: unknown[] = [];
		if (name !== undefined) { fields.push("name = ?"); values.push(name); }
		if (room_id !== undefined) { fields.push("room_id = ?"); values.push(room_id); }
		if (uuid !== undefined) { fields.push("uuid = ?"); values.push(uuid); }
		if (fields.length === 0) return c.json({ error: "nothing to update" }, 400);

		values.push(id);
		const row = await c.env.DB.prepare(
			`UPDATE streamers SET ${fields.join(", ")} WHERE id = ? RETURNING id, name, room_id, uuid`,
		)
			.bind(...values)
			.first();
		if (!row) return c.json({ error: "not found" }, 404);
		return row;
	}
}

export class StreamerDelete extends OpenAPIRoute {
	schema = {
		tags: ["Admin"],
		summary: "Delete streamer (and its sessions)",
		request: { params: z.object({ id: z.coerce.number().int() }) },
		responses: { "200": { description: "Deleted" } },
	};

	async handle(c: AppContext) {
		const data = await this.getValidatedData<typeof this.schema>();
		const { id } = data.params;
		const res = await c.env.DB.batch([
			c.env.DB.prepare("DELETE FROM sessions WHERE streamer_id = ?").bind(id),
			c.env.DB.prepare("DELETE FROM streamers WHERE id = ?").bind(id),
		]);
		if ((res[1]?.meta.changes ?? 0) === 0) return c.json({ error: "not found" }, 404);
		return { deleted: true };
	}
}
