/** Abstract provider base class and provider registry contracts. */

import {
	type LoomNetworkError,
	LoomProtocolError,
	toLoomNetworkError,
} from "./errors";
import { PackageLogger } from "./shared/logging";
import type {
	ConnectionMap,
	ConnectionOptions,
	ConnectionStateResult,
	NetworkType,
	ResolvedConnectionOptions,
	Result,
	URLOptions,
} from "./types/types";
import {
	ConnectionState,
	connectionOptionsSchema,
	LogLevel,
} from "./types/types";

/** Abstract transport provider base for all concrete implementations. */
export abstract class Provider<T extends NetworkType> {
	/** Parsed connection options with defaults applied. */
	protected options: ResolvedConnectionOptions;
	/** Logger target name derived from concrete class name. */
	protected readonly name: string;
	/** Current lifecycle state of this provider. */
	private currentState: ConnectionState = ConnectionState.IDLE;

	/**
	 * Creates a provider instance and validates initial options.
	 *
	 * @param options - Raw connection options for this provider.
	 */
	constructor(options: ConnectionOptions) {
		this.options = connectionOptionsSchema.parse(options);
		this.name = this.constructor.name;
	}

	/** Connects provider resources and updates lifecycle state. */
	abstract connect(options?: ConnectionOptions): Promise<ConnectionStateResult>;
	/** Closes provider resources and updates lifecycle state. */
	abstract close(): Promise<ConnectionStateResult>;
	/** Re-establishes provider resources after close/failure. */
	abstract reconnect(): Promise<ConnectionStateResult>;
	/** Returns concrete connection API surface for this provider. */
	abstract connection(): ConnectionMap[T];

	/** Builds full URL from configured scheme, host, path index, and query params. */
	protected buildURL(pathIndex = 0): string {
		const targetPath = this.options.url.paths[pathIndex];
		if (!targetPath) throw new Error(`invalid pathIndex ${pathIndex}`);
		const normalized = targetPath.startsWith("/")
			? targetPath
			: `/${targetPath}`;
		const base = new URL(
			`${this.options.url.scheme}://${this.options.url.host}${normalized}`,
		);
		const params = this.options.url.params ?? {};
		for (const [key, value] of Object.entries(params))
			base.searchParams.set(key, value);
		return base.toString();
	}

	/** Replaces and validates provider options. */
	protected setOptions(options: ConnectionOptions): ResolvedConnectionOptions {
		this.options = connectionOptionsSchema.parse(options);
		return this.options;
	}

	/** Returns currently resolved provider options. */
	protected getOptions(): ResolvedConnectionOptions {
		return this.options;
	}

	/** Returns currently resolved URL options. */
	protected getURL(): URLOptions {
		return this.options.url;
	}

	/** Applies next state and returns normalized transition result. */
	protected setState(
		nextState: ConnectionState,
		message: string,
	): ConnectionStateResult {
		const previousState = this.currentState;
		this.currentState = nextState;
		return {
			state: nextState,
			previousState,
			changed: nextState !== previousState,
			timestamp: new Date().toISOString(),
			message,
		};
	}

	/** Validates restricted state transitions before applying state. */
	protected ensureStateAllowed(
		nextState: ConnectionState,
	): ConnectionStateResult {
		const current = this.currentState;
		if (
			nextState === ConnectionState.CONNECTING &&
			current === ConnectionState.CLOSING
		) {
			throw new LoomProtocolError("cannot connect while closing");
		}
		if (
			nextState === ConnectionState.CLOSING &&
			current === ConnectionState.CONNECTING
		) {
			throw new LoomProtocolError("cannot close while connecting");
		}
		return this.setState(nextState, `state changed to ${nextState}`);
	}

	/** Returns current provider connection state. */
	state(): ConnectionState {
		return this.currentState;
	}

	/** Writes debug-level log line when configured level allows it. */
	protected logDebug(message: string, details = ""): boolean {
		const level = this.getOptions().logLevel;
		if (!PackageLogger.canLog(level, LogLevel.DEBUG)) return false;
		return PackageLogger.log(LogLevel.DEBUG, this.name, message, details);
	}

	/** Writes info-level log line when configured level allows it. */
	protected logInfo(message: string, details = ""): boolean {
		const level = this.getOptions().logLevel;
		if (!PackageLogger.canLog(level, LogLevel.INFO)) return false;
		return PackageLogger.log(LogLevel.INFO, this.name, message, details);
	}

	/** Writes warn-level log line when configured level allows it. */
	protected logWarn(message: string, details = ""): boolean {
		const level = this.getOptions().logLevel;
		if (!PackageLogger.canLog(level, LogLevel.WARN)) return false;
		return PackageLogger.log(LogLevel.WARN, this.name, message, details);
	}

	/** Writes error-level log line when configured level allows it. */
	protected logError(message: string, details = ""): boolean {
		const level = this.getOptions().logLevel;
		if (!PackageLogger.canLog(level, LogLevel.ERROR)) return false;
		return PackageLogger.log(LogLevel.ERROR, this.name, message, details);
	}

	/** Non-throwing connect wrapper returning `Result`. */
	async connectResult(
		options?: ConnectionOptions,
	): Promise<Result<ConnectionStateResult, LoomNetworkError>> {
		try {
			const value = await this.connect(options);
			return { ok: true, value };
		} catch (error) {
			return { ok: false, error: toLoomNetworkError(error as Error | object) };
		}
	}

	/** Non-throwing close wrapper returning `Result`. */
	async closeResult(): Promise<
		Result<ConnectionStateResult, LoomNetworkError>
	> {
		try {
			const value = await this.close();
			return { ok: true, value };
		} catch (error) {
			return { ok: false, error: toLoomNetworkError(error as Error | object) };
		}
	}

	/** Non-throwing reconnect wrapper returning `Result`. */
	async reconnectResult(): Promise<
		Result<ConnectionStateResult, LoomNetworkError>
	> {
		try {
			const value = await this.reconnect();
			return { ok: true, value };
		} catch (error) {
			return { ok: false, error: toLoomNetworkError(error as Error | object) };
		}
	}
}

/** Registry that maps each network type to a concrete provider class. */
export type ProviderRegistry = {
	[T in NetworkType]: new (
		options: ConnectionOptions,
	) => Provider<T>;
};
