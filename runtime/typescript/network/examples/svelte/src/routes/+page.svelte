<script lang="ts">
	import {
		runGraphqlDemo,
		runHttpDemo,
		runWebsocketDemo,
		type DemoResult
	} from "$lib/network-demo";

	type TabKey = "http" | "graphql" | "websocket";

	const tabs: { key: TabKey; label: string; description: string }[] = [
		{ key: "http", label: "HTTP", description: "GET request over HTTPS" },
		{ key: "graphql", label: "GraphQL", description: "Query over GraphQL HTTP endpoint" },
		{ key: "websocket", label: "WebSocket", description: "Send and listen over websocket" }
	];

	let activeTab = $state<TabKey>("http");
	let loading = $state(false);
	let result = $state<DemoResult | null>(null);

	async function runActive(): Promise<void> {
		loading = true;
		result = null;
		try {
			if (activeTab === "http") result = await runHttpDemo();
			else if (activeTab === "graphql") result = await runGraphqlDemo();
			else result = await runWebsocketDemo();
		} finally {
			loading = false;
		}
	}
</script>

<main class="page">
	<h1>loom-network transport examples</h1>
	<p class="subtitle">
		Select a provider tab, run the request, and inspect request, meta, and response payload.
	</p>

	<div class="tabs">
		{#each tabs as tab}
			<button
				type="button"
				class:active={activeTab === tab.key}
				onclick={() => {
					activeTab = tab.key;
					result = null;
				}}
			>
				{tab.label}
			</button>
		{/each}
	</div>

	<section class="panel">
		<h2>{tabs.find((item) => item.key === activeTab)?.label}</h2>
		<p>{tabs.find((item) => item.key === activeTab)?.description}</p>
		<button type="button" class="run" disabled={loading} onclick={runActive}>
			{loading ? "Running..." : "Run example"}
		</button>
	</section>

	{#if result}
		<section class="result">
			<h3>{result.name} result</h3>
			<p class:ok={result.ok} class:error={!result.ok}>
				{result.ok ? "Success" : "Failure"}
			</p>
			<pre><code>{JSON.stringify({ request: result.request }, null, 2)}</code></pre>
			<pre><code>{JSON.stringify({ meta: result.meta }, null, 2)}</code></pre>
			<pre><code>{JSON.stringify({ response: result.response, error: result.error }, null, 2)}</code></pre>
		</section>
	{/if}
</main>

<style>
	.page {
		max-width: 900px;
		margin: 0 auto;
		padding: 2rem 1rem 4rem;
		font-family: Inter, system-ui, sans-serif;
	}
	.subtitle {
		color: #475569;
	}
	.tabs {
		display: flex;
		gap: 0.5rem;
		margin: 1.2rem 0;
	}
	.tabs button {
		border: 1px solid #cbd5e1;
		background: #f8fafc;
		color: #0f172a;
		padding: 0.55rem 0.9rem;
		border-radius: 0.5rem;
		cursor: pointer;
	}
	.tabs button.active {
		background: #0f172a;
		color: #f8fafc;
		border-color: #0f172a;
	}
	.panel {
		border: 1px solid #e2e8f0;
		border-radius: 0.75rem;
		padding: 1rem;
		background: #fff;
	}
	.run {
		margin-top: 0.8rem;
		padding: 0.6rem 1rem;
		border-radius: 0.5rem;
		border: 1px solid #0f172a;
		background: #0f172a;
		color: #fff;
		cursor: pointer;
	}
	.run:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}
	.result {
		margin-top: 1rem;
		border: 1px solid #e2e8f0;
		border-radius: 0.75rem;
		padding: 1rem;
		background: #f8fafc;
	}
	.ok {
		color: #065f46;
		font-weight: 600;
	}
	.error {
		color: #991b1b;
		font-weight: 600;
	}
	pre {
		background: #0f172a;
		color: #e2e8f0;
		padding: 0.8rem;
		border-radius: 0.5rem;
		overflow-x: auto;
	}
</style>
