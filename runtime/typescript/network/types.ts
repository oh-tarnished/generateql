/** Public package type entrypoint for `@machanirobotics/loom-network/types` exports. */

export type {
	/** Constructor options accepted by `NetworkClient`. */
	NetworkClientOptions,
} from "./lib/connection";
export type {
	ConnectionLifecycle,
	ConnectionMap,
	ConnectionOptions,
	ConnectionStateHandler,
	ConnectionStateResult,
	GraphQLConnection,
	GraphQLErrorDetail,
	GraphQLOperationType,
	GraphQLRequestObject,
	GraphQLResult,
	GraphQLSubscriptionHandler,
	HTTPConnection,
	HTTPGraphQLMeta,
	HTTPMethod,
	HTTPRequestObject,
	HTTPResponse,
	JsonObject,
	JsonPrimitive,
	JsonValue,
	MessageHandler,
	ResolvedConnectionOptions,
	Result,
	TransportConnection,
	TransportMeta,
	TransportResponse,
	URLOptions,
	WebSocketConnection,
	WebSocketMessage,
	WebSocketMeta,
	WebSocketSendRequest,
} from "./lib/types/types";
export {
	ConnectionState,
	connectionOptionsSchema,
	HTTPProtocol,
	httpMethodSchema,
	LogLevel,
	NetworkType,
	URLScheme,
	urlOptionsSchema,
} from "./lib/types/types";
