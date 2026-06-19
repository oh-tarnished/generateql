import {
	runGraphQLExample,
	runGraphQLSubscriptionExample,
} from "./graphql/index";
import { runHTTPExample } from "./http/index";
import { runWebSocketExample } from "./websockets/index";

function main(): Promise<boolean> {
	return runHTTPExample()
		.then((ok) => {
			if (!ok) return false;
			return runGraphQLExample();
		})
		.then((ok) => {
			if (!ok) return false;
			return runGraphQLSubscriptionExample();
		})
		.then((ok) => {
			if (!ok) return false;
			return runWebSocketExample();
		})
		.then((ok) => {
			if (ok) console.log("All examples completed");
			return ok;
		})
		.catch((error) => {
			const message =
				error instanceof Error ? error.message : "unexpected failure";
			console.error("Example execution failed:", message);
			process.exitCode = 1;
			return false;
		});
}

main().then((ok) => {
	if (!ok) process.exitCode = 1;
});
