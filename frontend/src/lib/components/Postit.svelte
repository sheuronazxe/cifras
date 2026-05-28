<script lang="ts">
	import type { Snippet } from "svelte";

	interface Props {
		content?: string;
		pending?: string;
		fixed?: boolean;
		onClear?: () => void;
		children?: Snippet;
	}

	let {
		content = "",
		pending = "",
		fixed = false,
		onClear,
		children,
	}: Props = $props();
</script>

<div class="postit" class:postit-fixed={fixed}>
	{#if onClear}
		<button
			class="clear-btn"
			disabled={!content}
			onclick={onClear}
			title="Limpiar">BORRAR</button
		>
	{/if}

	<div class="postit-content">
		{#if children}
			{@render children()}
		{:else}
			<span>
				{#if content || pending}
					{#if content}{content}{#if pending}{"\n"}{/if}{/if}
					{#if pending}<span class="pending-expr">{pending}</span
						>{/if}
				{:else}
					<span class="postit-placeholder"
						>Selecciona dos números y una operación</span
					>
				{/if}
			</span>
		{/if}
	</div>
</div>

<style>
	.postit {
		position: relative;
		display: flex;
		flex-direction: column;
		background: linear-gradient(170deg, #ffeb64 0%, #bf9b34 100%);
		padding: 1.5rem;
		color: var(--bg);
		font-family: 'Special Elite', cursive;
	}

	.postit-fixed {
		width: 20rem;
		height: 18rem;
		transform: rotate(-1deg);
	}

	.postit::before {
		content: "";
		position: absolute;
		top: -16px;
		left: 50%;
		transform: translateX(-50%) rotate(1.5deg);
		width: 80px;
		height: 32px;
		background: rgba(255, 255, 255, 0.2);
		backdrop-filter: blur(1px);
		-webkit-backdrop-filter: blur(2px);
		border-radius: 1px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.15);
		z-index: 20;
		pointer-events: none;
	}

	.postit-content {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		text-align: center;
		white-space: pre-wrap;
		font-size: 1.5rem;
		font-weight: 500;
		line-height: 2.5rem;
		width: 100%;
		height: 100%;
		transform: translateY(0.3em);
	}

	.clear-btn {
		position: absolute;
		top: 10px;
		right: 10px;
		cursor: pointer;
		z-index: 25;
		background: none;
		border: none;
		outline: none;
		font: inherit;
		color: #933;
	}

	.clear-btn:hover:not(:disabled) {
		text-decoration: underline;
		text-decoration-thickness: 2px;
		text-underline-offset: 6px;
	}

	.clear-btn:disabled {
		opacity: 0.3;
		cursor: not-allowed;
	}

	.postit-placeholder {
		opacity: 0.5;
	}

	.pending-expr {
		opacity: 0.6;
		font-style: italic;
	}
</style>
