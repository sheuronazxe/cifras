<script lang="ts">
	import { game } from "$lib/store.svelte";
	import { sound } from "$lib/sound.svelte";
	import ReadyButton from "./ReadyButton.svelte";

	const isMeReady = $derived(game.isMeReady);
	const progressPercent = $derived(game.progressPercent);
	const hasReadyPlayer = $derived(game.gameState.players.some((p) => p.isReady));
</script>

<div class="info-layout">
	<div class="lobby-header">
		<h2>Sala de espera</h2>
		<div class="players-count">
			<span class="count-number">{game.gameState.players.length}</span>
			<span class="count-label"
				>{game.gameState.players.length === 1
					? "Jugador"
					: "Jugadores"}</span
			>
		</div>
	</div>

	<div class="progress-container">
		<div class="progress-bar" style="width: {hasReadyPlayer ? progressPercent : 0}%"></div>
	</div>

	<ul class="player-list">
		{#each game.gameState.players as player}
			<li class="player-item" class:is-ready={player.isReady}>
				<div class="player-left">
					<span class="player-name">
						<span class="text-ellipsis" title={player.name}
							>{player.name}</span
						>
						{#if game.isMe(player)}
							<span class="me-tag">Tú</span>
						{/if}
					</span>
				</div>
				<div class="player-right">
					{#if player.isReady}
						<span class="ready-badge">Listo</span>
					{:else}
						<span class="waiting-badge">Esperando</span>
					{/if}
				</div>
			</li>
		{:else}
			<li class="empty-list">
				<span class="spinner"></span>
				Esperando conexiones...
			</li>
		{/each}
	</ul>

	<ReadyButton
		isReady={isMeReady}
		readyText="esperando..."
		unreadyText="estoy listo"
	/>
</div>

<style>
	.lobby-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		width: 100%;
	}

	.lobby-header h2 {
		margin: 0;
		font-size: 1.5rem;
		font-weight: 700;
		letter-spacing: -0.02em;
		color: var(--text-muted);
	}

	.players-count {
		display: flex;
		align-items: baseline;
		gap: 0.3rem;
		background: rgba(255, 255, 255, 0.05);
		padding: 0.5rem 0.75rem;
		border-radius: 8px;
		border: 1px solid rgba(255, 255, 255, 0.1);
	}

	.count-number {
		font-weight: 700;
		color: var(--accent);
		font-size: 1rem;
	}

	.count-label {
		font-size: 0.8rem;
		opacity: 0.6;
		font-weight: 500;
	}

	.player-list {
		list-style: none;
		padding: 0;
		margin: 0 0 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		width: 100%;
	}

	.player-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.85rem 1rem;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 10px;
		transition: all 0.2s ease;
		min-width: 0;
	}

	.player-left {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		min-width: 0;
		flex: 1;
	}

	.player-name {
		font-size: 0.95rem;
		font-weight: 500;
		color: var(--text-muted);
		transition: color 0.2s ease;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		min-width: 0;
		flex: 1;
	}
	.player-item.is-ready .player-name {
		color: var(--text);
	}

	.me-tag {
		font-size: 0.7rem;
		font-weight: 700;
		text-transform: uppercase;
		background: var(--accent-muted);
		color: var(--text);
		padding: 0.15rem 0.45rem;
		border-radius: 4px;
		letter-spacing: 0.05em;
		flex-shrink: 0;
	}

	.player-right {
		display: flex;
		align-items: center;
		flex-shrink: 0;
		margin-left: 0.75rem;
	}

	.ready-badge {
		font-size: 0.8rem;
		font-weight: 600;
		color: var(--success);
		background: color-mix(in oklab, var(--success), transparent 85%);
		padding: 0.2rem 0.6rem;
		border-radius: 6px;
		border: 1px solid color-mix(in oklab, var(--success), transparent 70%);
	}

	.waiting-badge {
		font-size: 0.8rem;
		font-weight: 500;
		opacity: 0.5;
		background: var(--bg-light);
		padding: 0.2rem 0.6rem;
		border-radius: 6px;
		border: 1px solid rgba(255, 255, 255, 0.05);
	}

	.empty-list {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 2.5rem 1rem;
		opacity: 0.6;
		font-size: 0.9rem;
		gap: 0.75rem;
		text-align: center;
	}

</style>
