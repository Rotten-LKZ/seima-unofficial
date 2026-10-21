import { OpenAPIRoute } from "chanfana";
import { z } from "zod";
import { type AppContext, Session } from "../types";

export class Query extends OpenAPIRoute {
	schema = {
		tags: ["Public"],
		summary: "Query live sessions overlapping a time range (UTC)",
		request: {
			query: z.object({
				room_id: z.string().describe("Bilibili room id"),
				startTime: z.coerce
					.number()
					.int()
					.describe("Range start, unix seconds (UTC)"),
				endTime: z.coerce
					.number()
					.int()
					.describe("Range end, unix seconds (UTC)"),
			}),
		},
		responses: {
			"200": {
				description: "Sessions overlapping [startTime, endTime]; end_time null = still live",
				content: {
					"application/json": {
						schema: z.object({
							room_id: z.string(),
							sessions: Session.array(),
						}),
					},
				},
			},
		},
	};

	async handle(c: AppContext) {
		const data = await this.getValidatedData<typeof this.schema>();
		const { room_id, startTime, endTime } = data.query;

		if (endTime < startTime) {
			return c.json({ error: "endTime must be >= startTime" }, 400);
		}

		// Convert unix seconds (UTC) to stored unix milliseconds (UTC)
		const startMs = startTime * 1000;
		const endMs = endTime * 1000;

		const { results } = await c.env.DB.prepare(
			`SELECT s.id, s.streamer_id, s.start_time, s.end_time
			 FROM sessions s
			 JOIN streamers st ON st.id = s.streamer_id
			 WHERE st.room_id = ?
			   AND s.start_time <= ?
			   AND (s.end_time IS NULL OR s.end_time >= ?)
			 ORDER BY s.start_time ASC`,
		)
			.bind(room_id, endMs, startMs)
			.all();

		return { room_id, sessions: results ?? [] };
	}
}
