import type { LoomNetworkError } from "../../lib/errors";
import type { Result } from "../../lib/types/types";

/** Unwraps a Result value or throws its typed error message. */
export function unwrapResult<TValue>(
	result: Result<TValue, LoomNetworkError>,
): TValue {
	if (!result.ok) throw new Error(result.error.message);
	return result.value;
}
