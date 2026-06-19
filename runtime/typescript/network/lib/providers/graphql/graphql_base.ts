/**
 * GraphQL provider lifecycle/runtime setup base module.
 */
import {
	ApolloClient,
	from,
	HttpLink,
	InMemoryCache,
} from "@apollo/client/core";
import type { Client } from "graphql-ws";
import { LoomConnectError, LoomProtocolError } from "../../errors";
import { Provider } from "../../provider";
import {
	type ConnectionOptions,
	ConnectionState,
	type ConnectionStateResult,
	type GraphQLResult,
	type JsonObject,
	type NetworkType,
	URLScheme,
} from "../../types/types";
import type { GraphQLExecutionContext } from "./graphql_execution";
import { GraphQLHTTPMetaTracker } from "./graphql_meta";
import { assertGraphQLConnectivity } from "./graphql_operations";
import { createSubscriptionClient } from "./graphql_subscriptions";

/** Shared GraphQL provider base implementing lifecycle and runtime setup. */
abstract class GraphQLBaseProvider extends Provider<NetworkType.GRAPHQL> {
	/** Active Apollo HTTP client instance when connected. */
	protected client: ApolloClient | null = null;
	/** Active `graphql-ws` client for subscriptions when connected. */
	protected subscriptionClient: Client | null = null;
	/** Request-scoped metadata tracker for status/header envelopes. */
	protected readonly metaTracker = new GraphQLHTTPMetaTracker();

	/** Establishes GraphQL HTTP/subscription clients and validates connectivity. */
	override async connect(
		options?: ConnectionOptions,
	): Promise<ConnectionStateResult> {
		this.ensureStateAllowed(ConnectionState.CONNECTING);
		if (options) this.setOptions(options);
		const current = this.getOptions();
		this.logInfo(
			"connect start",
			`host=${current.url.host} scheme=${current.url.scheme}`,
		);
		if (![URLScheme.HTTP, URLScheme.HTTPS].includes(current.url.scheme)) {
			this.logError("connect failed", `invalid scheme=${current.url.scheme}`);
			throw new LoomProtocolError(
				`invalid GraphQL scheme: ${current.url.scheme}`,
			);
		}
		const target = this.buildURL(0);
		const trackedFetch = this.metaTracker.createTrackedFetch();
		const requestIdLink = this.metaTracker.createRequestIdLink();
		this.client = new ApolloClient({
			link: from([
				requestIdLink,
				new HttpLink({
					uri: target,
					fetch: trackedFetch,
					headers: current.headers,
				}),
			]),
			cache: new InMemoryCache(),
			defaultOptions: {
				query: { fetchPolicy: "no-cache" },
				mutate: { fetchPolicy: "no-cache" },
			},
		});
		this.subscriptionClient = createSubscriptionClient(target, current.headers);
		const state = this.setState(ConnectionState.CONNECTED, "connected");
		try {
			if (!current.skipConnectivityCheck) {
				const probe = await this.probeConnectivity(
					current.graphQLConnectivityQuery,
				);
				await assertGraphQLConnectivity(probe);
			}
		} catch (error) {
			this.setState(ConnectionState.CLOSED, "connectivity check failed");
			if (error instanceof Error) throw new LoomConnectError(error.message);
			throw new LoomConnectError("graphql connectivity check failed");
		}
		this.logInfo("connect success", `url=${target}`);
		return state;
	}

	/** Closes GraphQL clients and clears tracked request metadata. */
	override async close(): Promise<ConnectionStateResult> {
		this.ensureStateAllowed(ConnectionState.CLOSING);
		if (!this.client)
			return this.setState(ConnectionState.CLOSED, "already closed");
		this.logDebug("close");
		this.client.stop();
		this.client = null;
		this.subscriptionClient?.dispose();
		this.subscriptionClient = null;
		this.metaTracker.clear();
		return this.setState(ConnectionState.CLOSED, "closed");
	}

	/** Reconnects by closing existing clients and reconnecting with options. */
	override async reconnect(): Promise<ConnectionStateResult> {
		this.logInfo("reconnect");
		await this.close();
		return this.connect(this.getOptions());
	}

	/** Builds execution context consumed by operation helper functions. */
	protected executionContext(): GraphQLExecutionContext {
		return {
			state: () => this.state(),
			requireClient: () => this.requireClient(),
			requireSubscriptionClient: () => this.requireSubscriptionClient(),
			metaTracker: this.metaTracker,
			logDebug: (action, detail) => this.logDebug(action, detail),
			logInfo: (action, detail) => this.logInfo(action, detail),
		};
	}

	/** Returns connected Apollo client or throws connect error. */
	protected requireClient(): ApolloClient {
		if (!this.client)
			throw new LoomConnectError("graphql client is not connected");
		return this.client;
	}

	/** Returns connected subscription client or throws connect error. */
	protected requireSubscriptionClient(): Client {
		if (!this.subscriptionClient) {
			throw new LoomConnectError(
				"graphql subscription client is not connected",
			);
		}
		return this.subscriptionClient;
	}

	/** Runs the configured connectivity probe query during connect. */
	protected abstract probeConnectivity(
		query: string,
	): Promise<GraphQLResult<JsonObject>>;
}

export { GraphQLBaseProvider };
