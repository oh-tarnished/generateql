import { NetworkClient } from "@machanirobotics/loom-network";
import {
	type JsonObject,
	NetworkType,
} from "@machanirobotics/loom-network/types";
import { logResponse } from "../shared/printing";

export function runGraphQLExample(): Promise<boolean> {
	const client = new NetworkClient(NetworkType.GRAPHQL, {
		url: { host: "rickandmortyapi.com", paths: ["/graphql"] },
		timeoutMs: 10_000,
	});

	return client
		.connect()
		.then((connected) => {
			if (!connected.ok) {
				console.error("GraphQL connect failed:", connected.error.message);
				return false;
			}
			const connection = connected.value;
			return connection
				.query<{ character: { id: string; name: string; status: string } }>(
					`
      query CharacterById($id: ID!) {
        character(id: $id) {
          id
          name
          status
        }
      }
    `,
					{ id: "1" },
				)
				.then((result) => {
					if (!result.ok) {
						console.error("GraphQL query failed:", result.error.message);
						return false;
					}
					logResponse(result.value);
					return true;
				})
				.finally(() => {
					connection.closeResult().then(() => true);
				});
		})
		.catch((error) => {
			const message =
				error instanceof Error ? error.message : "graphql failure";
			console.error("GraphQL example failed:", message);
			return false;
		});
}

export function runGraphQLSubscriptionExample(): Promise<boolean> {
	const enabled = (process.env.LOOM_RUN_GQL_SUBSCRIPTION ?? "false") === "true";
	if (!enabled) {
		console.log(
			"GraphQL subscription example skipped (set LOOM_RUN_GQL_SUBSCRIPTION=true to enable)",
		);
		return Promise.resolve(true);
	}
	const host = process.env.LOOM_GQL_SUBSCRIPTION_HOST ?? "localhost:4000";
	const path = process.env.LOOM_GQL_SUBSCRIPTION_PATH ?? "/graphql";
	const document =
		process.env.LOOM_GQL_SUBSCRIPTION_DOCUMENT ??
		"subscription LoomExampleSubscription { __typename }";
	const client = new NetworkClient(NetworkType.GRAPHQL, {
		url: { host, paths: [path] },
		timeoutMs: 10_000,
		skipConnectivityCheck: true,
	});

	return client
		.connect()
		.then((connected) => {
			if (!connected.ok) {
				console.warn("GraphQL connect failed:", connected.error.message);
				return false;
			}
			const connection = connected.value;
			return connection
				.subscription<JsonObject>(document, (response) => {
					console.log("GraphQL subscription event");
					logResponse(response);
				})
				.then((stream) => {
					if (!stream.ok) {
						console.warn(
							"GraphQL subscription setup failed:",
							stream.error.message,
						);
						return false;
					}
					return Bun.sleep(1_000)
						.then(() => {
							stream.value();
							return true;
						})
						.finally(() => {
							connection.closeResult().then(() => true);
						});
				});
		})
		.catch((error) => {
			const message =
				error instanceof Error ? error.message : "graphql subscription failure";
			console.warn("GraphQL subscription example skipped:", message);
			return true;
		});
}

if (import.meta.main) {
	runGraphQLExample()
		.then((ok) => (ok ? runGraphQLSubscriptionExample() : false))
		.then((ok) => {
			if (!ok) process.exitCode = 1;
		});
}
