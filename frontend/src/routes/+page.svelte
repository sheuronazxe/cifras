<script lang="ts">
	import { game } from "$lib/store.svelte";
	import { sound } from "$lib/sound.svelte";
	import LoginForm from "$lib/components/LoginForm.svelte";
	import Lobby from "$lib/components/Lobby.svelte";
	import Choosing from "$lib/components/Choosing.svelte";
	import CifrasGame from "$lib/components/CifrasGame.svelte";
	import LetrasGame from "$lib/components/LetrasGame.svelte";
	import Results from "$lib/components/Results.svelte";
	import ToastContainer from "$lib/components/ToastContainer.svelte";

	function toggleMute() {
		sound.toggle();
		if (sound.enabled) {
			sound.playClick();
		}
	}

	$effect(() => {
		const endTime = game.gameState.endTime;
		const serverTime = game.gameState.serverTime;
		if (endTime > 0) {
			const clockOffset = serverTime > 0 ? serverTime - Date.now() : 0;
			let lastVal = -1;

			const update = () => {
				const currentServerTime = Date.now() + clockOffset;
				const newVal = Math.max(
					0,
					Math.floor((endTime - currentServerTime) / 1000),
				);

				if (newVal !== lastVal) {
					game.timeRemaining = newVal;
					if (
						newVal <= 10 &&
						newVal > 0 &&
						game.gameState.state === "PLAYING" &&
						lastVal !== -1
					) {
						sound.playTick();
					}
					lastVal = newVal;
				}
			};
			update();
			const interval = setInterval(update, 1000);
			return () => clearInterval(interval);
		} else {
			game.timeRemaining = 0;
		}
	});

	$effect(() => {
		return () => {
			game.disconnect();
		};
	});
</script>

<svelte:head>
	<title>Cifras y Letras — Multijugador en tiempo real</title>
	<meta
		name="description"
		content="Demuestra tu agilidad mental en este juego multijugador de Cifras y Letras."
	/>
	<meta name="theme-color" content="#050510" />
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="" />
	<link
		href="https://fonts.googleapis.com/css2?family=Roboto:ital,wght@0,100..900;1,100..900&family=Special+Elite&display=swap"
		rel="stylesheet"
	/>
</svelte:head>

<div class="layout">
	<header>
		<span class:connected={game.state === "CONNECTED"}>
			{game.state === "CONNECTED" ? " CONECTADO" : " DESCONECTADO"}
		</span>
		<button
			class="mute-btn"
			onclick={toggleMute}
			aria-label={sound.enabled ? "Silenciar" : "Activar sonido"}
		>
			{#if sound.enabled}
				<svg
					viewBox="0 0 24 24"
					width="18"
					height="18"
					fill="currentColor"
				>
					<path
						d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"
					/>
				</svg>
			{:else}
				<svg
					viewBox="0 0 24 24"
					width="18"
					height="18"
					fill="currentColor"
				>
					<path
						d="M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.21.05-.42.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51C20.63 14.91 21 13.5 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z"
					/>
				</svg>
			{/if}
		</button>
	</header>

	<main>
		{#if game.state !== "CONNECTED"}
			<LoginForm />
		{:else}
			{#key game.gameState.state}
				{#if game.gameState.state === "LOBBY"}
					<Lobby />
				{:else if game.gameState.state === "CHOOSING"}
					<Choosing />
				{:else if game.gameState.state === "PLAYING"}
					{#if game.gameState.roundType === "CIFRAS"}
						<CifrasGame />
					{:else}
						<LetrasGame />
					{/if}
				{:else if game.gameState.state === "FINISHED"}
					<Results />
				{/if}
			{/key}
		{/if}
		<ToastContainer />
	</main>
</div>

<style>
	.layout {
		max-width: 45rem;
		width: 100%;
		margin-inline: auto;
		min-height: 100dvh;
		display: grid;
		grid-template-rows: auto 1fr;
		background-color: var(--bg);
		border-inline: 1px solid var(--bg-light);
	}

	main {
		padding: 1rem;
		display: flex;
		flex-direction: column;
		justify-content: center;
		align-items: center;
	}

	header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		width: 100%;
		color: rgba(255, 255, 255, 0.5);
		font-size: 0.8rem;
		font-weight: bold;
		padding: 1rem;
	}

	.connected {
		color: var(--success);
	}

	.mute-btn {
		color: inherit;
		cursor: pointer;
		display: flex;
		align-items: center;
		background-color: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 0.4rem;
		padding: 0.4rem;
	}

	.mute-btn:hover {
		color: var(--text);
		background-color: rgba(255, 255, 255, 0.2);
	}
</style>
