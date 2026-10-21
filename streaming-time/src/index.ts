import { fromHono } from "chanfana";
import { Hono } from "hono";
import { adminAuth } from "./adminAuth";
import { corsLoli } from "./cors";
import {
	SessionCreate,
	SessionDelete,
	SessionList,
	SessionUpdate,
} from "./endpoints/adminSessions";
import {
	StreamerCreate,
	StreamerDelete,
	StreamerList,
	StreamerUpdate,
} from "./endpoints/adminStreamers";
import { GetInfo } from "./endpoints/getInfo";
import { Query } from "./endpoints/query";
import { fetchRoomInfo, parseLiveTime } from "./types";

const app = new Hono<{ Bindings: Env }>();

app.use("*", corsLoli);
app.use("/admin/*", adminAuth);

const openapi = fromHono(app, {
	docs_url: "/",
});

// Public endpoints
openapi.get("/getInfo", GetInfo);
openapi.get("/query", Query);

// Admin endpoints (bearer token)
openapi.get("/admin/streamers", StreamerList);
openapi.post("/admin/streamers", StreamerCreate);
openapi.put("/admin/streamers/:id", StreamerUpdate);
openapi.delete("/admin/streamers/:id", StreamerDelete);
openapi.get("/admin/sessions", SessionList);
openapi.post("/admin/sessions", SessionCreate);
openapi.put("/admin/sessions/:id", SessionUpdate);
openapi.delete("/admin/sessions/:id", SessionDelete);

export default {
	fetch: app.fetch,

	async scheduled(
		_event: ScheduledEvent,
		env: Env,
		ctx: ExecutionContext,
	): Promise<void> {
		ctx.waitUntil(pollStreamers(env));
	},
};

/**
 * Poll Bilibili for every tracked streamer and update session rows.
 * All stored times are unix milliseconds (UTC); bilibili live_time is unix
 * seconds (UTC) so it is multiplied by 1000 without any timezone conversion.
 */
async function pollStreamers(env: Env): Promise<void> {
	const { results: streamers } = await env.DB.prepare(
		"SELECT id, room_id FROM streamers",
	).all<{ id: number; room_id: string }>();

	const now = Date.now(); // unix ms, UTC

	await Promise.all(
		(streamers ?? []).map(async (streamer) => {
			try {
				const info = await fetchRoomInfo(streamer.room_id);
				const live = info.code === 0 && info.data?.live_status === 1;

				const open = await env.DB.prepare(
					"SELECT id FROM sessions WHERE streamer_id = ? AND end_time IS NULL ORDER BY start_time DESC LIMIT 1",
				)
					.bind(streamer.id)
					.first<{ id: number }>();

				if (live && !open) {
					// live_time is a CST ("+08:00") string; parseLiveTime returns unix ms (UTC)
					const startMs = parseLiveTime(info.data?.live_time) ?? now;
					await env.DB.prepare(
						"INSERT INTO sessions (streamer_id, start_time, end_time) VALUES (?, ?, NULL)",
					)
						.bind(streamer.id, startMs)
						.run();
				} else if (!live && open) {
					await env.DB.prepare(
						"UPDATE sessions SET end_time = ? WHERE id = ?",
					)
						.bind(now, open.id)
						.run();
				}
			} catch (e) {
				console.error(`poll failed for room ${streamer.room_id}:`, e);
			}
		}),
	);
}
