/**
 * Package-standard error types and normalization helpers.
 *
 * @remarks All transport/provider failures should be represented with these errors.
 */
/**
 * Base package error for all transport failures.
 *
 * @param message - Human-readable failure reason.
 */
export class LoomNetworkError extends Error {
	constructor(message: string) {
		super(message);
		this.name = "LoomNetworkError";
	}
}

/**
 * Connection establishment and reachability error.
 *
 * @param message - Human-readable failure reason.
 */
export class LoomConnectError extends LoomNetworkError {
	constructor(message: string) {
		super(message);
		this.name = "LoomConnectError";
	}
}

/**
 * Protocol/state-contract violation error.
 *
 * @param message - Human-readable failure reason.
 */
export class LoomProtocolError extends LoomNetworkError {
	constructor(message: string) {
		super(message);
		this.name = "LoomProtocolError";
	}
}

/**
 * Timeout error for operations exceeding configured timeout.
 *
 * @param message - Human-readable failure reason.
 */
export class LoomTimeoutError extends LoomNetworkError {
	constructor(message: string) {
		super(message);
		this.name = "LoomTimeoutError";
	}
}

/**
 * Converts unknown runtime errors into package-standard `LoomNetworkError`.
 *
 * @param error - Runtime error value.
 * @returns Existing `LoomNetworkError` or a normalized wrapper instance.
 */
export function toLoomNetworkError(error: Error | object): LoomNetworkError {
	if (error instanceof LoomNetworkError) return error;
	if (error instanceof Error) return new LoomNetworkError(error.message);
	return new LoomNetworkError("unknown network error");
}
