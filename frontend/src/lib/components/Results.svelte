<script lang="ts">
	import { game, type PlayerResult } from "$lib/store.svelte";
	import { sound } from "$lib/sound.svelte";
	import { fade } from "svelte/transition";
	import Postit from "./Postit.svelte";
	import ReadyButton from "./ReadyButton.svelte";

	const isMeReady = $derived(game.isMeReady);
	const progressPercent = $derived(game.progressPercent);
	const hasReadyPlayer = $derived(game.gameState.players.some((p) => p.isReady));

	function openDictionary(word: string | undefined) {
		sound.playClick();
		if (word) {
			window.open(
				`https://dle.rae.es/${encodeURIComponent(word.trim().toLowerCase())}`,
				"_blank",
			);
		}
	}

	const sortedRoundResults = $derived.by(() => {
		const list: PlayerResult[] = [];

		for (const p of game.gameState.players) {
			const res = game.gameState.otherResults?.find(
				(r) => r.playerId === p.id,
			);
			if (res) {
				list.push({ ...res });
			} else {
				list.push({
					playerId: p.id,
					name: p.name,
					points: 0,
					finalNumber: undefined,
					distance: undefined,
					word: undefined,
					expression: undefined,
				});
			}
		}

		list.sort((a, b) => {
			const scoreA =
				game.gameState.players.find((p) => p.id === a.playerId)?.score ?? 0;
			const scoreB =
				game.gameState.players.find((p) => p.id === b.playerId)?.score ?? 0;
			if (scoreA !== scoreB) {
				return scoreB - scoreA;
			}
			if (a.points !== b.points) {
				return b.points - a.points;
			}
			if (game.gameState.roundType === "CIFRAS") {
				const distA = a.distance !== undefined ? a.distance : 999999;
				const distB = b.distance !== undefined ? b.distance : 999999;
				if (distA !== distB) {
					return distA - distB;
				}
			} else if (game.gameState.roundType === "LETRAS") {
				const lenA = a.word ? a.word.length : 0;
				const lenB = b.word ? b.word.length : 0;
				if (lenA !== lenB) {
					return lenB - lenA;
				}
			}
			return a.name.localeCompare(b.name);
		});

		return list;
	});
</script>

<div class="info-layout" transition:fade>
	<div class="winner-announcement">
		{#if game.gameState.winner === "Nadie"}
			Nadie ha puntuado en esta ronda
		{:else if game.gameState.winner === "Empate"}
			¡Empate! Varios jugadores han ganado
		{:else}
			<strong>{game.gameState.winner}</strong> ha ganado la ronda
		{/if}{#if game.gameState.roundType === "CIFRAS" && game.gameState.exactSolutionSteps?.length > 0 && !game.gameState.otherResults?.some((r) => r.distance === 0)}, sin embargo se podía hacer exacto.{/if}
	</div>

	<div class="results-postit">
		{#if game.gameState.roundType === "CIFRAS"}
			{@const steps = game.gameState.exactSolutionSteps?.length
				? game.gameState.exactSolutionSteps
				: game.gameState.solutionSteps}

			{#if steps?.length > 0}
				<Postit
					fixed={true}
					content={steps
						.join("\n")
						.replace(/\*/g, "×")
						.replace(/\//g, "÷")}
				/>
			{/if}
		{:else if game.gameState.exactSolutionSteps?.length > 0}
			<Postit fixed={true}>
				<div class="postit-words-list">
					{#each game.gameState.exactSolutionSteps.slice(0, 5) as word}
						<button
							class="postit-word-link"
							onclick={() => openDictionary(word)}
							title="Consultar en el diccionario de la RAE"
						>
							{word}
						</button>
					{/each}
				</div>
			</Postit>
		{/if}
	</div>

	<div class="progress-container">
		<div class="progress-bar" style="width: {hasReadyPlayer ? progressPercent : 0}%"></div>
	</div>

	<div class="table-container">
		<div class="results-table">
			{#if sortedRoundResults.length > 0}
				{#each sortedRoundResults as res}
					<div
						class="grid-row"
					class:me={game.me?.name === res.name}
					>
						<span class="col-name player-row-name">
							<span class="text-ellipsis" title={res.name}
								>{res.name}</span
							>
						</span>
						<span class="col-sol player-row-val">
							{#if game.gameState.roundType === "CIFRAS"}
								{#if res.finalNumber !== undefined}
									{res.finalNumber}
									<small>
										&nbsp;({res.distance === 0
											? "exacto"
											: (res.distance ?? "-")})
									</small>
								{:else}
									<span class="no-submission">-</span>
								{/if}
							{:else if res.word}
								<button
									class="word-link table-word"
									onclick={() => openDictionary(res.word)}
									title="Consultar en el diccionario de la RAE"
								>
									{res.word}
								</button>
							{:else}
								<span class="no-submission">-</span>
							{/if}
						</span>
						<span
							class="col-pts points-cell"
							class:zero-points={res.points === 0}
							>+{res.points}</span
						>
						<span class="col-total total-cell">
							{game.gameState.players.find(
								(p) => p.id === res.playerId,
							)?.score ?? 0}
						</span>
						<span class="col-status status-cell">
							<span
								class="status-dot"
								class:is-ready={game.gameState.players.find(
									(p) => p.id === res.playerId,
								)?.isReady}
							></span>
						</span>
					</div>
				{/each}
			{:else}
				<div class="empty-row">No hay jugadores</div>
			{/if}
		</div>
	</div>

	<ReadyButton 
		isReady={isMeReady} 
		readyText="esperando..." 
		unreadyText="estoy listo" 
	/>
</div>

<style>
	.winner-announcement {
		font-weight: 300;
		opacity: 0.7;
		margin: 0;
		text-align: center;
		text-wrap: balance;
	}
	.winner-announcement strong {
		color: var(--accent);
		font-weight: 600;
	}

	.postit-words-list {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		width: 100%;
	}
	.postit-word-link {
		background: none;
		border: none;
		padding: 0;
		color: var(--postit-text);
		font-family: 'Special Elite', cursive;
		font-size: 1.45rem;
		font-weight: 700;
		text-transform: uppercase;
		cursor: pointer;
		text-decoration: underline dashed var(--postit-text);
		text-underline-offset: 4px;
		transition: color 0.2s, text-decoration-color 0.2s;
		margin: 0.25rem 0;
	}
	.postit-word-link:hover {
		color: var(--bg);
		text-decoration-style: solid;
	}

	.table-container {
		width: 100%;
		max-height: 400px;
		overflow-y: auto;
	}

	.results-table {
		display: grid;
		grid-template-columns: 1fr 1fr max-content max-content 32px;
		row-gap: 0.4rem;
		width: 100%;
		margin: 0;
	}

	.grid-row {
		display: grid;
		grid-column: 1 / -1;
		grid-template-columns: subgrid;
		background: rgba(255, 255, 255, 0.05);
		border-radius: 8px;
		transition: transform 0.2s ease, background 0.2s ease, border-color 0.2s ease;
		align-items: center;
		border: 1px solid rgba(255, 255, 255, 0.1);
	}

	.grid-row.me {
		background: color-mix(in oklab, var(--accent), transparent 85%);
	}

	.grid-row > span {
		padding: 0.75rem 1rem;
		display: flex;
		align-items: center;
	}

	.col-pts {
		justify-self: center;
	}

	.col-total {
		justify-self: center;
	}

	.col-status {
		width: 32px;
		justify-self: center;
	}

	.status-cell {
		padding-left: 0 !important;
		justify-content: center;
	}

	.player-row-name {
		font-weight: 500;
		color: var(--text);
		gap: 0.75rem;
		min-width: 0;
	}

	.status-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
		background: var(--text-muted);
		transition: all 0.3s ease;
		box-shadow: 0 0 0 0 rgba(255, 255, 255, 0);
	}

	.status-dot.is-ready {
		background: var(--success);
		box-shadow: 0 0 10px color-mix(in oklab, var(--success), transparent 40%);
	}

	.player-row-val {
		font-weight: 400;
		opacity: 0.9;
	}

	.points-cell {
		font-weight: bold;
		color: var(--success);
	}

	.total-cell {
		font-weight: 800;
		color: var(--accent);
	}

	.empty-row {
		grid-column: 1 / -1;
		text-align: center;
		opacity: 0.4;
		padding: 3rem;
		background: var(--bg-light);
		border-radius: 8px;
	}

	.results-postit {
		width: 224px;
		height: 196px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.results-postit :global(.postit) {
		transform: scale(0.7);
		transform-origin: center center;
		flex-shrink: 0;
		margin: 0;
	}

	.word-link {
		background: transparent;
		border: none;
		color: var(--accent);
		font-family: inherit;
		font-size: inherit;
		padding: 0;
		cursor: pointer;
		text-decoration: underline;
		text-decoration-style: dashed;
		text-underline-offset: 4px;
		transition: color 0.2s ease, text-decoration-color 0.2s ease;
	}
	.word-link:hover {
		color: var(--accent);
		text-decoration-style: solid;
	}
	.table-word {
		font-weight: 500;
	}
	.no-submission {
		opacity: 0.4;
		font-weight: 300;
	}
	.points-cell.zero-points {
		opacity: 0.4;
		font-weight: normal;
	}

	@media (max-width: 480px) {
		.results-table {
			grid-template-columns: 1.2fr 1fr max-content max-content 24px;
		}
		.grid-row > span {
			padding: 0.5rem 0.3rem;
			font-size: 0.85rem;
		}
		.col-status {
			width: 24px;
		}
	}
</style>