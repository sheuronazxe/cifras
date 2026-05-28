<script lang="ts">
	import { sound } from '$lib/sound.svelte';
	import { game } from '$lib/store.svelte';

	interface Props {
		isReady: boolean;
		readyText: string;
		unreadyText: string;
	}

	let {
		isReady,
		readyText,
		unreadyText,
	}: Props = $props();

	function handleReady() {
		sound.playClick();
		game.ready();
	}
</script>

<button
	class="custom-button"
	class:ready={isReady}
	onclick={handleReady}
>
	{isReady ? readyText : unreadyText}
</button>

<style>
	.ready {
		background-color: var(--success);
		color: var(--text);
	}

	.ready:hover {
		background-color: var(--success);
		box-shadow: 0 1px 10px color-mix(in oklab, var(--success), transparent 50%);
	}
</style>