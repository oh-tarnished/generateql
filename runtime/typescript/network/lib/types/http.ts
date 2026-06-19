/**
 * HTTP transport request/response and connection contract definitions.
 */

import type { z } from "zod";
import type { LoomNetworkError } from "../errors";
import type { HTTPProtocol } from "./enums";
import type { JsonValue } from "./json";
import type { Result, TransportConnection } from "./lifecycle";
import type { HTTPMethod } from "./schemas";

/** Shared metadata envelope for HTTP and GraphQL responses. */
export interface HTTPGraphQLMeta<TTransport extends "http" | "graphql"> {
	/** HTTP status code returned by the server. */
	status: number;
	/** Lower-cased response headers map. */
	headers: Record<string, string>;
	/** Transport discriminator used for narrowing. */
	transport: TTransport;
}

type TDataOfSchema = JsonValue;

/** Single HTTP request object accepted by the provider. */
export interface HTTPRequestObject {
	/** Request method verb. */
	method: HTTPMethod;
	/** Optional direct path that overrides `pathIndex`. */
	path?: string;
	/** Path index into configured URL paths. */
	pathIndex?: number;
	/** Optional JSON request body. */
	body?: JsonValue;
	/** Optional request-specific headers merged over connection defaults. */
	headers?: Record<string, string>;
	/** Optional retry override for this request only. */
	maxRetries?: number;
	/** Optional timeout override in milliseconds. */
	timeoutMs?: number;
	/** Optional abort signal propagated to fetch. */
	signal?: AbortSignal;
	/** Optional zod schema used to validate and narrow response data. */
	responseSchema?: z.ZodType<TDataOfSchema>;
}

/** Normalized HTTP response envelope. */
export interface HTTPResponse<TData extends JsonValue = JsonValue> {
	/** Unified metadata envelope for transport details. */
	meta: HTTPGraphQLMeta<"http">;
	/** HTTP status code returned by the server. */
	status: number;
	/** Native HTTP success flag (`status` in range 200-299). */
	ok: boolean;
	/** Response headers map. */
	headers: Record<string, string>;
	/** Parsed response payload, optionally schema-validated. */
	data: TData;
	/** Requested protocol preference (`auto`, `http1`, or `http3`). */
	protocol: HTTPProtocol;
	/** Raw response content type header. */
	contentType: string;
	/** Non-transport parsing/validation errors for this envelope. */
	errors: string[];
}

/** HTTP transport contract exposed by the package. */
export interface HTTPConnection
	extends TransportConnection<HTTPRequestObject, HTTPResponse<JsonValue>> {
	/** Executes one HTTP request with retries and timeout support. */
	request<TData extends JsonValue = JsonValue>(
		request: HTTPRequestObject,
	): Promise<Result<HTTPResponse<TData>, LoomNetworkError>>;
}
