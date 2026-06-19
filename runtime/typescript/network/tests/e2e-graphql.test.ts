import { describe, expect, test } from "bun:test";
import { NetworkClient } from "../lib";
import { NetworkType } from "../lib/types/types";
import { unwrapResult } from "./helpers/result";

describe("GraphQL e2e", () => {
	test("connect and query return normalized GraphQL envelope", async () => {
		const client = new NetworkClient(NetworkType.GRAPHQL, {
			url: { host: "rickandmortyapi.com", paths: ["/graphql"] },
			timeoutMs: 10_000,
		});
		const connection = unwrapResult(await client.connect());

		const result = await connection.query<{
			character: { id: string; name: string };
		}>(
			`
      query CharacterById($id: ID!) {
        character(id: $id) {
          id
          name
        }
      }
      `,
			{ id: "1" },
		);
		await connection.close();

		const response = unwrapResult(result);
		expect(response.meta.transport).toBe("graphql");
		expect(response.meta.status).toBe(200);
		expect(response.networkError).toBeNull();
		expect(response.errors.length).toBe(0);
		expect(response.data?.character.id).toBe("1");
	});

	test("invalid GraphQL query returns normalized GraphQL errors", async () => {
		const client = new NetworkClient(NetworkType.GRAPHQL, {
			url: { host: "rickandmortyapi.com", paths: ["/graphql"] },
			timeoutMs: 10_000,
		});
		const connection = unwrapResult(await client.connect());

		const result = await connection.query("{ invalidField }");
		await connection.close();

		const response = unwrapResult(result);
		expect(response.meta.transport).toBe("graphql");
		expect(response.errors.length > 0 || response.networkError !== null).toBe(
			true,
		);
	});
});
