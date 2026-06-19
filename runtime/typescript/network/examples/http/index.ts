import { NetworkClient } from "@machanirobotics/loom-network";
import { HTTPProtocol, NetworkType } from "@machanirobotics/loom-network/types";
import { logResponse } from "../shared/printing";

export function runHTTPExample(): Promise<boolean> {
	const client = new NetworkClient(NetworkType.HTTPS, {
		url: { host: "jsonplaceholder.typicode.com", paths: ["/todos/1"] },
		timeoutMs: 8_000,
		retries: 1,
		httpProtocol: HTTPProtocol.HTTP1,
	});

	return client
		.connect()
		.then((connected) => {
			if (!connected.ok) {
				console.error("HTTP connect failed:", connected.error.message);
				return false;
			}
			console.log("HTTP connected via Result wrapper");
			const connection = connected.value;
			return connection
				.request<{ id: number; title: string; completed: boolean }>({
					method: "GET",
					path: "/todos/1",
				})
				.then((response) => {
					if (!response.ok) {
						console.error("HTTP request failed:", response.error.message);
						return false;
					}
					logResponse(response.value);
					return true;
				})
				.finally(() => {
					connection.closeResult().then(() => true);
				});
		})
		.catch((error) => {
			const message = error instanceof Error ? error.message : "http failure";
			console.error("HTTP example failed:", message);
			return false;
		});
}

if (import.meta.main) {
	runHTTPExample().then((ok) => {
		if (!ok) process.exitCode = 1;
	});
}
