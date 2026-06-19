/**
 * HTTP request execution helpers with retry/timeout/normalization behavior.
 */
import { LoomConnectError, LoomTimeoutError } from "../../errors";
import type {
	HTTPRequestObject,
	HTTPResponse,
	JsonValue,
	ResolvedConnectionOptions,
	URLOptions,
} from "../../types/types";

/** Runtime hooks injected by `HttpProvider` for request execution. */
interface HTTPRequestHooks {
	/** Logs debug events for request lifecycle traces. */
	logDebug: (action: string, detail?: string) => boolean;
	/** Logs warning events (typically retries). */
	logWarn: (action: string, detail?: string) => boolean;
}

/** Converts response headers to plain object map. */
function responseHeadersToObject(headers: Headers): Record<string, string> {
	const output: Record<string, string> = {};
	for (const [key, value] of headers.entries()) output[key] = value;
	return output;
}

/** Creates timeout-aware abort signal merged with optional external signal. */
function createSignal(
	external: AbortSignal | undefined,
	timeoutMs: number,
): AbortSignal {
	if (!external) return AbortSignal.timeout(timeoutMs);
	const controller = new AbortController();
	const timeout = setTimeout(() => controller.abort(), timeoutMs);
	external.addEventListener("abort", () => controller.abort(), { once: true });
	controller.signal.addEventListener("abort", () => clearTimeout(timeout), {
		once: true,
	});
	return controller.signal;
}

/** Builds full URL from arbitrary path while preserving base query params. */
function buildURLFromPath(url: URLOptions, path: string): string {
	const normalized = path.startsWith("/") ? path : `/${path}`;
	const base = new URL(`${url.scheme}://${url.host}${normalized}`);
	const params = url.params ?? {};
	for (const [key, value] of Object.entries(params))
		base.searchParams.set(key, value);
	return base.toString();
}

/** Executes one HTTP request with retries, timeout, and optional schema parsing. */
async function performHTTPRequest<TData extends JsonValue = JsonValue>(
	request: HTTPRequestObject,
	options: ResolvedConnectionOptions,
	url: string,
	retryDelayMs: number,
	hooks: HTTPRequestHooks,
): Promise<HTTPResponse<TData>> {
	const body = request.body ?? null;
	const mergedHeaders = { ...options.headers, ...(request.headers ?? {}) };
	const maxRetries = request.maxRetries ?? options.retries;
	const timeoutMs = request.timeoutMs ?? options.timeoutMs;
	let attempt = 0;
	hooks.logDebug(
		"request start",
		`method=${request.method} pathIndex=${String(request.pathIndex ?? 0)} retries=${maxRetries}`,
	);
	while (true) {
		try {
			const signal = createSignal(request.signal, timeoutMs);
			if (body !== null) {
				mergedHeaders["Content-Type"] =
					mergedHeaders["Content-Type"] ?? "application/json";
			}
			const response = await fetch(url, {
				method: request.method,
				headers: mergedHeaders,
				signal,
				body: body === null ? undefined : JSON.stringify(body),
			});
			const text = await response.text();
			const contentType = response.headers.get("content-type") ?? "";
			const parsedValue = contentType.includes("application/json")
				? (JSON.parse(text || "null") as JsonValue)
				: (text as JsonValue);
			const parsed = request.responseSchema
				? (request.responseSchema.parse(parsedValue) as TData)
				: (parsedValue as TData);
			const headers = responseHeadersToObject(response.headers);
			return {
				meta: { status: response.status, headers, transport: "http" },
				status: response.status,
				ok: response.ok,
				headers,
				data: parsed,
				protocol: options.httpProtocol,
				contentType,
				errors: [],
			};
		} catch (error) {
			if (attempt >= maxRetries) {
				if (error instanceof Error && error.name === "AbortError") {
					throw new LoomTimeoutError(`request timeout after ${timeoutMs}ms`);
				}
				if (error instanceof Error) throw new LoomConnectError(error.message);
				throw new LoomConnectError("request failed");
			}
			hooks.logWarn(
				"request retry",
				`method=${request.method} attempt=${attempt + 1}`,
			);
			attempt += 1;
			await Bun.sleep(retryDelayMs);
		}
	}
}

export { buildURLFromPath, performHTTPRequest };
