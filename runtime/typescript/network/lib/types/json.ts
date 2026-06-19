/**
 * JSON primitive/object/value type aliases used by transport payload contracts.
 */
/** Primitive JSON values supported by transport payloads. */
export type JsonPrimitive = string | number | boolean | null;

/** Recursive JSON value type used for request/response data. */
export type JsonValue = JsonPrimitive | JsonObject | JsonValue[];

/** JSON object map with string keys and JSON-compatible values. */
export type JsonObject = { [key: string]: JsonValue };
