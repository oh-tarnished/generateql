/**
 * Cross-transport union contracts and connection map definitions.
 */
import { NetworkType } from "./enums";
import type { GraphQLConnection, GraphQLResult } from "./graphql";
import type { HTTPConnection, HTTPGraphQLMeta, HTTPResponse } from "./http";
import type { JsonObject, JsonValue } from "./json";
import type {
	WebSocketConnection,
	WebSocketMessage,
	WebSocketMeta,
} from "./websocket";

/** Unified response envelope for all supported transports. */
export type TransportResponse<
	THTTPData extends JsonValue = JsonValue,
	TGraphQLData extends JsonObject = JsonObject,
> = HTTPResponse<THTTPData> | GraphQLResult<TGraphQLData> | WebSocketMessage;

/** Unified metadata envelope for all supported transports. */
export type TransportMeta =
	| HTTPGraphQLMeta<"http">
	| HTTPGraphQLMeta<"graphql">
	| WebSocketMeta;

/** Maps transport type discriminators to their concrete connection contracts. */
export type ConnectionMap = {
	[NetworkType.HTTP]: HTTPConnection;
	[NetworkType.HTTPS]: HTTPConnection;
	[NetworkType.WEBSOCKET]: WebSocketConnection;
	[NetworkType.GRAPHQL]: GraphQLConnection;
};
