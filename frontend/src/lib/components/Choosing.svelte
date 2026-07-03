<script lang="ts">
	import { game } from '$lib/store.svelte';
	import { sound } from '$lib/sound.svelte';

	function chooseVowels(count: number) {
		sound.playClick();
		game.chooseVowels(count);
	}
</script>

<div class="center-content">
	{#if game.isChooser}
		<h2>¡Te toca elegir!</h2>
		<p>¿Cuántas vocales quieres?</p>
		<div class="buttons-row">
			<button class="custom-button vowel-btn" onclick={() => chooseVowels(3)}>3</button>
			<button class="custom-button vowel-btn" onclick={() => chooseVowels(4)}>4</button>
			<button class="custom-button vowel-btn" onclick={() => chooseVowels(5)}>5</button>
			<button class="custom-button vowel-btn" onclick={() => chooseVowels(6)}>6</button>
		</div>
	{:else}
		<h2>Le toca a {game.gameState.chooser}</h2>
		<p>Esperando a que elija las letras...</p>
		<div class="loader"></div>
	{/if}
</div>

<style>
	.center-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		text-align: center;
		gap: 1.5rem;
	}

	.center-content h2 {
		margin: 0;
		font-size: 1.8rem;
		color: var(--text);
	}

	.center-content p {
		margin: 0;
		opacity: 0.6;
	}

	.buttons-row {
		display: flex;
		gap: 1rem;
		justify-content: center;
	}

	.vowel-btn {
		width: 60px;
		font-size: 1.2rem;
		padding: 0.75rem 0;
		font-family: 'Roboto', sans-serif;
	}

	.center-content :global(.loader) {
		margin: 0;
	}
</style>