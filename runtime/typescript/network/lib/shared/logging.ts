/**
 * Shared structured logger utilities for package runtime diagnostics.
 *
 * @remarks Provides Go-style single-line logs with optional ANSI coloring.
 */
import { LogLevel } from "../types/types";

const LEVEL_WEIGHT: Record<LogLevel, number> = {
	[LogLevel.SILENT]: 100,
	[LogLevel.ERROR]: 40,
	[LogLevel.WARN]: 30,
	[LogLevel.INFO]: 20,
	[LogLevel.DEBUG]: 10,
};

type PackageMeta = {
	name: string;
	version: string;
};

type LoggerRuntimeConfig = {
	packageName: string;
	version: string;
	environment: string;
	colorMode: "auto" | "always" | "never";
};

const ANSI = {
	reset: "\x1b[0m",
	dim: "\x1b[2m",
	gray: "\x1b[90m",
	white: "\x1b[97m",
	cyan: "\x1b[36m",
	magenta: "\x1b[35m",
	green: "\x1b[32m",
	yellow: "\x1b[33m",
	red: "\x1b[31m",
	blue: "\x1b[34m",
} as const;

const defaultMeta = loadDefaultMeta();
let runtimeConfig: LoggerRuntimeConfig = {
	packageName: defaultMeta.name,
	version: defaultMeta.version,
	environment: resolveEnvironment(),
	colorMode: resolveColorMode(),
};

/** Extracts source filename and line number from current stack trace. */
function resolveCallerLocation(): string {
	const stackRaw = new Error().stack ?? "";
	const lines = stackRaw.split("\n").map((line) => line.trim());
	for (const line of lines) {
		if (!line.includes(":")) continue;
		if (line.includes("lib/logging.ts") || line.includes("logging.ts"))
			continue;
		if (!line.includes(".ts")) continue;
		const match = line.match(/((\/|[A-Za-z]:\\).+?):(\d+):(\d+)/);
		if (!match) continue;
		const filePath = match[1] ?? "";
		const lineNumber = match[3] ?? "0";
		return `${basename(filePath)}:${lineNumber}`;
	}
	return "unknown:0";
}

/** Returns local timestamp in ISO-like format with timezone offset. */
function timestampWithOffset(): string {
	const date = new Date();
	const y = date.getFullYear();
	const m = `${date.getMonth() + 1}`.padStart(2, "0");
	const d = `${date.getDate()}`.padStart(2, "0");
	const hh = `${date.getHours()}`.padStart(2, "0");
	const mm = `${date.getMinutes()}`.padStart(2, "0");
	const ss = `${date.getSeconds()}`.padStart(2, "0");
	const offsetMinutes = -date.getTimezoneOffset();
	const sign = offsetMinutes >= 0 ? "+" : "-";
	const abs = Math.abs(offsetMinutes);
	const offH = `${Math.floor(abs / 60)}`.padStart(2, "0");
	const offM = `${abs % 60}`.padStart(2, "0");
	return `${y}-${m}-${d}T${hh}:${mm}:${ss}${sign}${offH}:${offM}`;
}

/** Resolves ANSI color for a log level. */
function levelColor(level: LogLevel): string {
	if (level === LogLevel.ERROR) return ANSI.red;
	if (level === LogLevel.WARN) return ANSI.yellow;
	if (level === LogLevel.INFO) return ANSI.green;
	if (level === LogLevel.DEBUG) return ANSI.gray;
	return ANSI.white;
}

/** Returns final path segment from absolute/relative file path. */
function basename(pathValue: string): string {
	const parts = pathValue.split(/[\\/]/);
	const last = parts[parts.length - 1] ?? "unknown";
	return last;
}

/** Reads environment variable safely across supported runtimes. */
function readEnv(key: string): string {
	const source = globalThis as { process?: { env?: Record<string, string> } };
	const value = source.process?.env?.[key];
	return value ?? "";
}

/** Returns true when running in browser-like global scope. */
function isBrowser(): boolean {
	const scope = globalThis as { window?: object; document?: object };
	return Boolean(scope.window) && Boolean(scope.document);
}

/** Loads package metadata defaults from environment overrides. */
function loadDefaultMeta(): PackageMeta {
	const nameFromEnv = readEnv("LOOM_PACKAGE_NAME");
	const versionFromEnv = readEnv("LOOM_PACKAGE_VERSION");
	return {
		name: nameFromEnv || "@machanirobotics/loom-network",
		version: versionFromEnv || "0.0.0",
	};
}

/** Resolves environment label for log output. */
function resolveEnvironment(): string {
	const env = readEnv("NODE_ENV");
	return env || "development";
}

/** Resolves color mode from environment (`auto`, `always`, `never`). */
function resolveColorMode(): "auto" | "always" | "never" {
	const mode = readEnv("LOOM_COLOR");
	if (mode === "always") return "always";
	if (mode === "never") return "never";
	return "auto";
}

/** Builds a plain or ANSI-colored log line depending on runtime settings. */
function colorizeLine(
	stamp: string,
	level: LogLevel,
	levelText: string,
	source: string,
	env: string,
	target: string,
	payload: string,
	packageName: string,
	packageVersion: string,
): string {
	const colorMode = runtimeConfig.colorMode;
	const shouldColor =
		colorMode === "always" || (colorMode === "auto" && !isBrowser());
	if (!shouldColor) {
		return `${stamp} ${levelText} <${source}> ${packageName} (${packageVersion} | ${env}): [${target}] ${payload}`;
	}
	const lvlColor = levelColor(level);
	const ts = `${ANSI.dim}${stamp}${ANSI.reset}`;
	const lvl = `${lvlColor}${levelText}${ANSI.reset}`;
	const src = `${ANSI.cyan}<${source}>${ANSI.reset}`;
	const pkg = `${ANSI.magenta}${packageName}${ANSI.reset}`;
	const meta = `${ANSI.gray}(${packageVersion} | ${env})${ANSI.reset}`;
	const tag = `${ANSI.blue}[${target}]${ANSI.reset}`;
	const body = `${ANSI.white}${payload}${ANSI.reset}`;
	return `${ts} ${lvl} ${src} ${pkg} ${meta}: ${tag} ${body}`;
}

/**
 * Shared package logger with Go-style log line formatting.
 *
 * @remarks Works in Bun/Node and browser runtimes. Color can be controlled with `LOOM_COLOR`.
 */
export const PackageLogger = {
	/** Writes one formatted log line at the provided log level. */
	log(level: LogLevel, target: string, message: string, details = ""): boolean {
		const stamp = timestampWithOffset();
		const source = resolveCallerLocation();
		const env = runtimeConfig.environment;
		const levelText = level.toUpperCase();
		const payload = details ? `${message} ${details}` : message;
		const line = colorizeLine(
			stamp,
			level,
			levelText,
			source,
			env,
			target,
			payload,
			runtimeConfig.packageName,
			runtimeConfig.version,
		);
		if (level === LogLevel.ERROR) console.error(line);
		else if (level === LogLevel.WARN) console.warn(line);
		else console.log(line);
		return true;
	},

	/** Returns true when `incoming` should be emitted for `current` level. */
	canLog(current: LogLevel, incoming: LogLevel): boolean {
		return (
			LEVEL_WEIGHT[incoming] >= LEVEL_WEIGHT[current] &&
			current !== LogLevel.SILENT
		);
	},

	/** Overrides runtime logger metadata and color settings. */
	configure(options: Partial<LoggerRuntimeConfig>): LoggerRuntimeConfig {
		runtimeConfig = { ...runtimeConfig, ...options };
		return runtimeConfig;
	},
};
