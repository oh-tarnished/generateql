/**
 * HTTP provider implementation module.
 */

import {
	type LoomNetworkError,
	LoomProtocolError,
	toLoomNetworkError,
} from "../../errors";
import { Provider } from "../../provider";
import {
	type ConnectionOptions,
	ConnectionState,
	type ConnectionStateResult,
	type HTTPConnection,
	type HTTPRequestObject,
	type HTTPResponse,
	type JsonValue,
	type NetworkType,
	type Result,
} from "../../types/types";
import {
	probeHTTPConnectivity,
	validateHTTPConnectOptions,
} from "./http_connect";
import { buildURLFromPath, performHTTPRequest } from "./http_request";

/**
 * HTTP transport provider using `fetch` with retries, timeout, and schema parsing.
 */
export class HttpProvider
	extends Provider<NetworkType.HTTP>
	implements HTTPConnection
{
	/** Validates connectivity and transitions provider state to connected. */
	override async connect(
		options?: ConnectionOptions,
	): Promise<ConnectionStateResult> {
		this.ensureStateAllowed(ConnectionState.CONNECTING);
		if (options) this.setOptions(options);
		const current = this.getOptions();
		this.logInfo(
			"connect start",
			`host=${current.url.host} scheme=${current.url.scheme} protocol=${current.httpProtocol}`,
		);
		validateHTTPConnectOptions(current);
		const target = this.buildURL(0);
		await probeHTTPConnectivity(current, target);
		return this.setState(ConnectionState.CONNECTED, "connected");
	}

	/** Closes the HTTP provider state (no persistent socket to tear down). */
	override async close(): Promise<ConnectionStateResult> {
		this.ensureStateAllowed(ConnectionState.CLOSING);
		this.logDebug("close");
		return this.setState(ConnectionState.CLOSED, "closed");
	}

	/** Reconnects by resetting state and running connectivity check again. */
	override async reconnect(): Promise<ConnectionStateResult> {
		this.logInfo("reconnect");
		await this.close();
		return this.connect(this.getOptions());
	}

	/** Returns the concrete HTTP connection interface. */
	override connection(): HTTPConnection {
		return this;
	}

	/** Generic execute alias for HTTP request method. */
	async execute<TData extends JsonValue = JsonValue>(
		request: HTTPRequestObject,
	): Promise<Result<HTTPResponse<TData>, LoomNetworkError>> {
		return this.request<TData>(request);
	}

	/** Executes one HTTP request with retry and timeout behavior. */
	async request<TData extends JsonValue = JsonValue>(
		request: HTTPRequestObject,
	): Promise<Result<HTTPResponse<TData>, LoomNetworkError>> {
		try {
			if (this.state() !== ConnectionState.CONNECTED) {
				throw new LoomProtocolError("request requires connected state");
			}
			const options = this.getOptions();
			const url = request.path
				? buildURLFromPath(this.getURL(), request.path)
				: this.buildURL(request.pathIndex ?? 0);
			const value = await performHTTPRequest<TData>(
				request,
				options,
				url,
				options.retryDelayMs,
				{
					logDebug: (action, detail) => this.logDebug(action, detail),
					logWarn: (action, detail) => this.logWarn(action, detail),
				},
			);
			return { ok: true, value };
		} catch (error) {
			return { ok: false, error: toLoomNetworkError(error as Error | object) };
		}
	}
}
