<script lang="ts">
	import { game } from "$lib/store.svelte";
	import { sound } from "$lib/sound.svelte";
	
	let loginName = $state(game.playerName);
	let isShaking = $state(false);

	function joinGame(e: Event) {
		e.preventDefault();
		const trimmedName = loginName.trim();
		if (trimmedName.length === 0) {
			isShaking = true;
			sound.playError();
			setTimeout(() => (isShaking = false), 400);
			return;
		}
		sound.playClick();
		game.connect(trimmedName);
	}
</script>

{#if game.state === "CONNECTING" || game.state === "RECONNECTING"}
	<div class="loader"></div>
{:else}
	<div class="login-layout" class:shake={isShaking}>
		{#if game.state === "DISCONNECTED"}
			<p>Se perdió la conexión con el servidor.</p>
			<button
				class="custom-button"
				onclick={() => {
					sound.playClick();
					game.reconnect();
				}}
			>
				Reconectar
			</button>
		{:else}
			<h1 class="logo-title">Cifras & Letras</h1>
			<p class="subtitle">DEMUESTRA TU AGILIDAD MENTAL</p>
			<form onsubmit={joinGame}>
				<input
					id="login-name"
					type="text"
					class="custom-input"
					bind:value={loginName}
					placeholder="Tu nombre"
					spellcheck="false"
					maxlength="20"
					aria-label="Tu nombre"
				/>
				<button type="submit" class="custom-button"
					>entrar</button
				>
			</form>
		{/if}
	</div>
{/if}

<style>
	.login-layout {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		text-align: center;
	}

	h1 {
		font-size: 2.2rem;
		margin: 0 0 0.5rem 0;
		font-weight: 700;
		color: var(--text);
	}

	.logo-title {
		font-family: 'Special Elite', cursive;
		letter-spacing: 0.05em;
	}

	.subtitle {
		font-size: 0.8rem;
		letter-spacing: 0.2em;
		opacity: 0.6;
		margin: 0 0 2.5rem 0;
	}

	p {
		font-size: 1rem;
		opacity: 0.6;
		margin: 0 0 1.5rem 0;
	}

	form {
		width: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 1rem;
	}
</style>
