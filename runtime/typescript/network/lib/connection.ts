/**
 * Network client factory and provider registry module.
 *
 * @remarks Exposes `NetworkClient`, the main runtime API for consumers.
 */

import {
	type LoomNetworkError,
	LoomProtocolError,
	toLoomNetworkError,
} from "./errors";
import type { ProviderRegistry } from "./provider";
import { GraphQLProvider } from "./providers/graphql/graphql";
import { HttpProvider } from "./providers/http/http";
import { WebSocketProvider } from "./providers/websockets/websockets";
import { PackageLogger } from "./shared/logging";
import type {
	ConnectionMap,
	ConnectionOptions,
	Result,
	URLOptions,
} from "./types/types";
import { LogLevel, NetworkType, URLScheme } from "./types/types";

/**
 * Maps each `NetworkType` to the concrete provider implementation.
 *
 * @remarks
 * This registry is the factory lookup used by `NetworkClient.connect()`.
 */
const providerRegistry: ProviderRegistry = {
	[NetworkType.HTTP]: HttpProvider,
	[NetworkType.HTTPS]: HttpProvider,
	[NetworkType.WEBSOCKET]: WebSocketProvider,
	[NetworkType.GRAPHQL]: GraphQLProvider,
};

/**
 * Public options accepted by `NetworkClient` constructor.
 *
 * @remarks
 * URL scheme is optional and inferred from transport type when omitted.
 */
export type NetworkClientOptions = Omit<ConnectionOptions, "url"> & {
	url: Omit<URLOptions, "scheme"> & { scheme?: URLScheme };
};

/**
 * High-level factory that creates typed transport connections.
 *
 * @typeParam T - Transport kind to connect (`http`, `https`, `graphql`, `websocket`).
 */
export class NetworkClient<T extends NetworkType> {
	/** Selected transport type for this client instance. */
	private type: T;
	/** User-provided connection options (scheme resolved lazily). */
	private options: NetworkClientOptions;

	/**
	 * Creates a typed network client.
	 *
	 * @param type - Target transport family.
	 * @param options - Transport options excluding required URL scheme.
	 */
	constructor(type: T, options: NetworkClientOptions) {
		this.type = type;
		this.options = options;
	}

	/**
	 * Creates and connects the underlying provider.
	 *
	 * @returns Result wrapper containing either a typed connection or a typed error.
	 */
	async connect(): Promise<Result<ConnectionMap[T], LoomNetworkError>> {
		try {
			const logLevel = this.options.logLevel ?? LogLevel.INFO;
			const resolvedOptions = this.resolveOptions();
			if (PackageLogger.canLog(logLevel, LogLevel.INFO)) {
				PackageLogger.log(
					LogLevel.INFO,
					"NetworkClient",
					"creating provider",
					`type=${this.type}`,
				);
			}
			const ProviderClass = providerRegistry[this.type];
			const provider = new ProviderClass(resolvedOptions);
			await provider.connect(resolvedOptions);
			if (PackageLogger.canLog(logLevel, LogLevel.INFO)) {
				PackageLogger.log(
					LogLevel.INFO,
					"NetworkClient",
					"provider connected",
					`type=${this.type}`,
				);
			}
			const value = provider.connection();
			return { ok: true, value };
		} catch (error) {
			return { ok: false, error: toLoomNetworkError(error as Error | object) };
		}
	}

	/**
	 * Resolves inferred scheme from transport type and validates compatibility.
	 *
	 * @returns Fully resolved connection options including URL scheme.
	 */
	private resolveOptions(): ConnectionOptions {
		const inferredScheme = this.schemeForType(this.type);
		const selectedScheme = this.options.url.scheme ?? inferredScheme;
		this.assertSchemeCompatibility(this.type, selectedScheme);
		return {
			...this.options,
			url: { ...this.options.url, scheme: selectedScheme },
		};
	}

	/**
	 * Infers default URL scheme for a transport type.
	 *
	 * @param type - Requested transport type.
	 * @returns Compatible default scheme.
	 */
	private schemeForType(type: NetworkType): URLScheme {
		if (type === NetworkType.HTTP) return URLScheme.HTTP;
		if (type === NetworkType.HTTPS || type === NetworkType.GRAPHQL)
			return URLScheme.HTTPS;
		return URLScheme.WSS;
	}

	/**
	 * Guards transport/scheme compatibility before provider creation.
	 *
	 * @param type - Requested transport type.
	 * @param scheme - Chosen URL scheme.
	 * @returns True when scheme is valid for the transport.
	 */
	private assertSchemeCompatibility(
		type: NetworkType,
		scheme: URLScheme,
	): boolean {
		if (type === NetworkType.HTTP && scheme !== URLScheme.HTTP) {
			throw new LoomProtocolError(
				`NetworkType.HTTP requires scheme ${URLScheme.HTTP}`,
			);
		}
		if (type === NetworkType.HTTPS && scheme !== URLScheme.HTTPS) {
			throw new LoomProtocolError(
				`NetworkType.HTTPS requires scheme ${URLScheme.HTTPS}`,
			);
		}
		if (
			type === NetworkType.GRAPHQL &&
			![URLScheme.HTTP, URLScheme.HTTPS].includes(scheme)
		) {
			throw new LoomProtocolError(
				"NetworkType.GRAPHQL requires http or https scheme",
			);
		}
		if (
			type === NetworkType.WEBSOCKET &&
			![URLScheme.WS, URLScheme.WSS].includes(scheme)
		) {
			throw new LoomProtocolError(
				"NetworkType.WEBSOCKET requires ws or wss scheme",
			);
		}
		return true;
	}
}
