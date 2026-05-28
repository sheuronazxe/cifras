<script lang="ts">
	import { game } from '$lib/store.svelte';
	import { fly } from 'svelte/transition';
</script>

<div class="toast-container">
	{#each game.toasts as toast (toast.id)}
		<div class="toast {toast.type}" transition:fly={{ y: 20, duration: 300 }}>
			{toast.message}
		</div>
	{/each}
</div>

<style>
	.toast-container {
		position: fixed;
		top: 2rem;
		left: 50%;
		transform: translateX(-50%);
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		z-index: 1000;
		pointer-events: none;
	}
	.toast {
		padding: 0.75rem 1.5rem;
		border-radius: 4px;
		background: var(--bg-light);
		border: 1px solid rgba(255, 255, 255, 0.15);
		box-shadow: 0 4px 12px rgba(0,0,0,0.3);
		color: var(--text);
		font-size: 0.9rem;
	}
	.toast.success { border-color: var(--success); color: var(--success); }
	.toast.error { border-color: var(--error); color: var(--error); }
</style>