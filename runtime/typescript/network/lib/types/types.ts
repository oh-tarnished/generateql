/**
 * Consolidated explicit re-export surface for all type modules.
 *
 * @remarks This file is the single import source for `lib/types/*` consumers.
 */
export {
	ConnectionState,
	HTTPProtocol,
	LogLevel,
	NetworkType,
	URLScheme,
} from "./enums";
export type {
	GraphQLConnection,
	GraphQLErrorDetail,
	GraphQLOperationType,
	GraphQLRequestObject,
	GraphQLResult,
	GraphQLSubscriptionHandler,
} from "./graphql";
export type {
	HTTPConnection,
	HTTPGraphQLMeta,
	HTTPRequestObject,
	HTTPResponse,
} from "./http";
export type { JsonObject, JsonPrimitive, JsonValue } from "./json";
export type {
	ConnectionLifecycle,
	ConnectionStateHandler,
	ConnectionStateResult,
	Result,
	TransportConnection,
} from "./lifecycle";
export type {
	ConnectionOptions,
	HTTPMethod,
	ResolvedConnectionOptions,
	URLOptions,
} from "./schemas";
export {
	connectionOptionsSchema,
	httpMethodSchema,
	urlOptionsSchema,
} from "./schemas";
export type {
	ConnectionMap,
	TransportMeta,
	TransportResponse,
} from "./transport";
export type {
	MessageHandler,
	WebSocketConnection,
	WebSocketMessage,
	WebSocketMeta,
	WebSocketSendRequest,
} from "./websocket";
