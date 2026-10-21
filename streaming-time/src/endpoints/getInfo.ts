import { OpenAPIRoute } from "chanfana";
import { z } from "zod";
import { type AppContext, fetchRoomInfo } from "../types";

export class GetInfo extends OpenAPIRoute {
	schema = {
		tags: ["Public"],
		summary: "Proxy Bilibili getRoomPlayInfo",
		request: {
			query: z.object({
				room_id: z.string().describe("Bilibili room id"),
			}),
		},
		responses: {
			"200": { description: "Upstream response" },
		},
	};

	async handle(c: AppContext) {
		const data = await this.getValidatedData<typeof this.schema>();
		try {
			const info = await fetchRoomInfo(data.query.room_id);
			return info;
		} catch (e) {
			return c.json({ error: (e as Error).message }, 502);
		}
	}
}
