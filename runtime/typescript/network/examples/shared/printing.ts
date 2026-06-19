import type {
	TransportMeta,
	TransportResponse,
} from "@machanirobotics/loom-network/types";

export function logMeta(meta: TransportMeta): boolean {
	console.log("Meta:", meta);
	return true;
}

export function logResponse(response: TransportResponse): boolean {
	logMeta(response.meta);
	console.log("Payload:", response.data);
	return true;
}
