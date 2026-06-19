/**
 * Zod-backed runtime schemas and inferred option/input type contracts.
 *
 * @remarks These schemas validate connection configuration before provider usage.
 */
import { z } from "zod";
import { HTTPProtocol, LogLevel, URLScheme } from "./enums";

/** Runtime-validated URL configuration schema. */
export const urlOptionsSchema = z.object({
	scheme: z.enum(URLScheme),
	host: z.string().min(1),
	paths: z.array(z.string().min(1)).min(1),
	params: z.record(z.string(), z.string()).optional(),
});

/** Runtime-validated connection configuration schema. */
export const connectionOptionsSchema = z.object({
	url: urlOptionsSchema,
	timeoutMs: z.number().int().positive().default(10_000),
	headers: z.record(z.string(), z.string()).default({}),
	retries: z.number().int().min(0).default(0),
	retryDelayMs: z.number().int().min(0).default(2_000),
	skipConnectivityCheck: z.boolean().default(false),
	graphQLConnectivityQuery: z.string().default("query { __typename }"),
	websocketReconnectDelayMs: z.number().int().positive().default(5_000),
	httpProtocol: z.enum(HTTPProtocol).default(HTTPProtocol.AUTO),
	logLevel: z.enum(LogLevel).default(LogLevel.INFO),
});

/** Supported HTTP methods for the request object. */
export const httpMethodSchema = z.enum([
	"GET",
	"POST",
	"PUT",
	"PATCH",
	"DELETE",
]);

/** Inferred method union from `httpMethodSchema`. */
export type HTTPMethod = z.infer<typeof httpMethodSchema>;

/** URL options accepted by the public API. */
export interface URLOptions extends z.infer<typeof urlOptionsSchema> {}

/** Input connection options accepted by constructors and connect calls. */
export interface ConnectionOptions
	extends z.input<typeof connectionOptionsSchema> {}

/** Fully-resolved connection options with defaults applied. */
export interface ResolvedConnectionOptions
	extends z.infer<typeof connectionOptionsSchema> {}
