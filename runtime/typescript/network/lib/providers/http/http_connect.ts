/**
 * HTTP connectivity validation helpers used during provider connect flow.
 */
import { LoomConnectError, LoomProtocolError } from "../../errors";
import { PackageLogger } from "../../shared/logging";
import {
	HTTPProtocol,
	LogLevel,
	type ResolvedConnectionOptions,
	URLScheme,
} from "../../types/types";

/** Logger target name used for provider-formatted HTTP connect logs. */
const HTTP_PROVIDER_TARGET = "HttpProvider";

/** Emits one connect-phase log line through shared package logger. */
function logHTTPConnect(
	options: ResolvedConnectionOptions,
	level: LogLevel,
	message: string,
	details = "",
): boolean {
	if (!PackageLogger.canLog(options.logLevel, level)) return false;
	return PackageLogger.log(level, HTTP_PROVIDER_TARGET, message, details);
}

/** Validates scheme/protocol compatibility before attempting connectivity checks. */
function validateHTTPConnectOptions(
	options: ResolvedConnectionOptions,
): boolean {
	if (![URLScheme.HTTP, URLScheme.HTTPS].includes(options.url.scheme)) {
		logHTTPConnect(
			options,
			LogLevel.ERROR,
			"connect failed",
			`invalid scheme=${options.url.scheme}`,
		);
		throw new LoomProtocolError(`invalid HTTP scheme: ${options.url.scheme}`);
	}
	if (
		options.httpProtocol === HTTPProtocol.HTTP3 &&
		options.url.scheme !== URLScheme.HTTPS
	) {
		logHTTPConnect(
			options,
			LogLevel.ERROR,
			"connect failed",
			"HTTP/3 requires HTTPS",
		);
		throw new LoomProtocolError("HTTP/3 requires HTTPS scheme");
	}
	return true;
}

/** Performs the runtime connectivity probe used by `HttpProvider.connect`. */
async function probeHTTPConnectivity(
	options: ResolvedConnectionOptions,
	target: string,
): Promise<boolean> {
	if (options.skipConnectivityCheck) {
		logHTTPConnect(
			options,
			LogLevel.INFO,
			"connect success",
			"connectivity check skipped",
		);
		return true;
	}
	let probeResponse = await fetch(target, {
		method: "HEAD",
		headers: options.headers,
	});
	if (probeResponse.status === 405) {
		probeResponse = await fetch(target, {
			method: "GET",
			headers: options.headers,
		});
	}
	if (!probeResponse.ok) {
		logHTTPConnect(
			options,
			LogLevel.ERROR,
			"connect failed",
			`connectivity check status=${probeResponse.status}`,
		);
		throw new LoomConnectError(
			`connectivity check failed with status ${probeResponse.status}`,
		);
	}
	if (options.httpProtocol === HTTPProtocol.HTTP3) {
		const altSvc = probeResponse.headers.get("alt-svc") ?? "";
		if (!altSvc.toLowerCase().includes("h3")) {
			logHTTPConnect(
				options,
				LogLevel.ERROR,
				"connect failed",
				"HTTP/3 requested but h3 not advertised",
			);
			throw new LoomConnectError(
				"HTTP/3 requested but server did not advertise h3 via alt-svc",
			);
		}
	}
	logHTTPConnect(
		options,
		LogLevel.INFO,
		"connect success",
		`status=${probeResponse.status} url=${target}`,
	);
	logHTTPConnect(
		options,
		LogLevel.WARN,
		"httpProtocol note",
		"HTTP/3 is preference/validation, runtime fetch may negotiate another protocol",
	);
	return true;
}

export { probeHTTPConnectivity, validateHTTPConnectOptions };
