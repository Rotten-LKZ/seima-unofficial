import type { Context } from "hono";
import { z } from "zod";

export type AppContext = Context<{ Bindings: Env }>;

export const Streamer = z.object({
	id: z.number().int(),
	name: z.string(),
	room_id: z.string(),
	uuid: z.string(),
});

export const Session = z.object({
	id: z.number().int(),
	streamer_id: z.number().int(),
	start_time: z.number().int().describe("Unix milliseconds"),
	end_time: z.number().int().nullable().describe("Unix milliseconds"),
});

export interface BilibiliRoomInfo {
	code: number;
	message: string;
	data: {
		room_id: number;
		uid: number;
		live_status: number;
		// e.g. "2026-10-21 08:27:37" in China Standard Time (UTC+8), or "0000-00-00 00:00:00" when offline
		live_time: string;
	} | null;
}

export async function fetchRoomInfo(roomId: string): Promise<BilibiliRoomInfo> {
	const url = `https://api.live.bilibili.com/room/v1/Room/get_info?room_id=${encodeURIComponent(roomId)}`;
	const res = await fetch(url, {
		headers: {
			"User-Agent":
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
			Referer: "https://live.bilibili.com/",
		},
	});
	if (!res.ok) throw new Error(`bilibili upstream ${res.status}`);
	return (await res.json()) as BilibiliRoomInfo;
}

/**
 * Parse bilibili live_time ("YYYY-MM-DD HH:mm:ss", CST = UTC+8) to unix ms (UTC).
 * Returns null when the value is missing or the zero placeholder.
 */
export function parseLiveTime(liveTime: string | undefined): number | null {
	if (!liveTime || liveTime === "0000-00-00 00:00:00") return null;
	return Date.parse(`${liveTime}+08:00`);
}
