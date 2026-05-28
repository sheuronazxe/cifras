<script lang="ts">
	import { game } from '$lib/store.svelte';
	import { sound } from '$lib/sound.svelte';
	import Postit from './Postit.svelte';

	let word = $state("");
	let lettersState = $state<{ id: number, char: string, used: boolean }[]>([]);
	let selectedIds = $state<number[]>([]);
	let previousRoundKey = $state('');

	function initRound(letters: string[]) {
		lettersState = letters.map((char, index) => ({
			id: index,
			char,
			used: false
		}));
		word = "";
		selectedIds = [];
	}

	$effect(() => {
		const state = game.gameState.state;
		const roundType = game.gameState.roundType;
		const letters = game.gameState.letters;
		const roundKey = game.roundKey;
		
		if (state === 'PLAYING' && roundType === 'LETRAS' && letters?.length > 0) {
			if (roundKey !== previousRoundKey) {
				initRound(letters);
				previousRoundKey = roundKey;
				game.myBestWord = null;
			}
		}
	});

	function selectLetter(letter: typeof lettersState[0]) {
		if (letter.used) return;
		sound.playClick();
		letter.used = true;
		word += letter.char;
		selectedIds = [...selectedIds, letter.id];
	}

	function undoLetter() {
		if (selectedIds.length > 0) {
			sound.playClick();
			const lastId = selectedIds[selectedIds.length - 1];
			selectedIds = selectedIds.slice(0, -1);
			word = word.slice(0, -1);
			const letter = lettersState.find(l => l.id === lastId);
			if (letter) {
				letter.used = false;
			}
		}
	}

	function clearWord() {
		sound.playClick();
		lettersState.forEach(l => {
			l.used = false;
		});
		selectedIds = [];
		word = "";
	}

	function shuffleLetters() {
		sound.playClick();
		let arr = [...lettersState];
		for (let i = arr.length - 1; i > 0; i--) {
			const j = Math.floor(Math.random() * (i + 1));
			[arr[i], arr[j]] = [arr[j], arr[i]];
		}
		lettersState = arr;
	}

	function submitWord() {
		sound.playClick();
		if (word.length >= 5) {
			game.submitLetras(word);
		} else {
			game.addToast("Mínimo 5 letras", "error");
		}
	}
</script>

<div class="game-layout letras-view">
	<div class="first-row">
		<div class="info-panel">
			<span class="info-label">Mi Palabra</span>
			<span class="info-value accented">{game.myBestWord || "- - -"}</span>
		</div>
		<div class="info-panel">
			<span class="info-label">Tiempo</span>
			<span class="info-value" class:urgent={game.timeRemaining <= 10}>{game.timeRemaining}</span>
		</div>
	</div>

	<div class="postit-container">
		<Postit content={word} onClear={clearWord}>
			<span class="composing-word">{word || "..."}</span>
		</Postit>
	</div>

	<div class="letters-row">
		{#each lettersState as letter}
			<button 
				class="letter-tile game-tile" 
				class:used={letter.used}
				aria-label="Seleccionar letra {letter.char}"
				onclick={() => selectLetter(letter)}
			>
				{letter.char}
			</button>
		{/each}
	</div>

	<div class="actions-row">
		<button class="action-btn undo-btn" onclick={undoLetter} disabled={selectedIds.length === 0}>Atrás</button>
		<button class="action-btn shuffle-btn" onclick={shuffleLetters} title="Mezclar letras">
			<svg fill="currentColor" width="22" height="22" viewBox="0 0 256 256" xmlns="http://www.w3.org/2000/svg" class="die-icon">
				<g fill-rule="evenodd">
					<path d="M47.895 88.097c.001-4.416 3.064-9.837 6.854-12.117l66.257-39.858c3.785-2.277 9.915-2.277 13.707.008l66.28 39.934c3.786 2.28 6.853 7.703 6.852 12.138l-.028 79.603c-.001 4.423-3.069 9.865-6.848 12.154l-66.4 40.205c-3.781 2.29-9.903 2.289-13.69-.01l-66.167-40.185c-3.78-2.295-6.842-7.733-6.84-12.151l.023-79.72zm13.936-6.474l65.834 36.759 62.766-36.278-62.872-36.918L61.83 81.623zM57.585 93.52c0 1.628-1.065 71.86-1.065 71.86-.034 2.206 1.467 4.917 3.367 6.064l61.612 37.182.567-77.413s-64.48-39.322-64.48-37.693zm76.107 114.938l60.912-38.66c2.332-1.48 4.223-4.915 4.223-7.679V93.125l-65.135 37.513v77.82z"/>
					<path d="M77.76 132.287c-4.782 2.762-11.122.735-14.16-4.526-3.037-5.261-1.622-11.765 3.16-14.526 4.783-2.762 11.123-.735 14.16 4.526 3.038 5.261 1.623 11.765-3.16 14.526zm32 21c-4.782 2.762-11.122.735-14.16-4.526-3.037-5.261-1.622-11.765 3.16-14.526 4.783-2.762 11.123-.735 14.16 4.526 3.038 5.261 1.623 11.765-3.16 14.526zm-32 16c-4.782 2.762-11.122.735-14.16-4.526-3.037-5.261-1.622-11.765 3.16-14.526 4.783-2.762 11.123-.735 14.16 4.526 3.038 5.261 1.623 11.765-3.16 14.526zm32 21c-4.782 2.762-11.122.735-14.16-4.526-3.037-5.261-1.622-11.765 3.16-14.526 4.783-2.762 11.123-.735 14.16 4.526 3.038 5.261 1.623 11.765-3.16 14.526zm78.238-78.052c-4.783-2.762-11.122-.735-14.16 4.526-3.037 5.261-1.623 11.765 3.16 14.526 4.783 2.762 11.123.735 14.16-4.526 3.038-5.261 1.623-11.765-3.16-14.526zm-16.238 29c-4.782-2.762-11.122-.735-14.16 4.526-3.037 5.261-1.622 11.765 3.16 14.526 4.783 2.762 11.123.735 14.16-4.526 3.038-5.261 1.623-11.765-3.16-14.526zm-17 28c-4.782-2.762-11.122-.735-14.16 4.526-3.037 5.261-1.622 11.765 3.16 14.526 4.783 2.762 11.123.735 14.16-4.526 3.038-5.261 1.623-11.765-3.16-14.526zM128.5 69c-6.351 0-11.5 4.925-11.5 11s5.149 11 11.5 11S140 86.075 140 80s-5.149-11-11.5-11z"/>
				</g>
			</svg>
		</button>
		<button class="action-btn send-btn" onclick={submitWord}>Enviar</button>
	</div>
</div>

<style>
	.postit-container {
		width: 100%;
		height: 140px;
	}

	.postit-container :global(.postit) {
		width: 100%;
		height: 100%;
	}

	.first-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1.5rem;
		width: 100%;
	}

	.composing-word {
		font-size: 2.8rem;
		font-weight: 800;
		letter-spacing: 0.3rem;
		text-transform: uppercase;
		font-family: 'Special Elite', cursive;
		color: var(--postit-text);
	}

	.letters-row {
		display: grid;
		grid-template-columns: repeat(10, 1fr);
		gap: 0.5rem;
		justify-content: center;
		width: 100%;
		max-width: 100%;
		margin: 0 auto;
		padding: 0.5rem 0;
	}

	.letter-tile {
		width: 100%;
		min-width: 0;
		aspect-ratio: 1 / 1;
		padding: 0;
		font-size: 1.5rem;
		font-weight: 700;
		text-transform: uppercase;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--text);
	}

	.letter-tile:hover:not(.used) {
		transform: translateY(-2px);
	}

	.letter-tile.used {
		opacity: 0.4;
		pointer-events: none;
	}

	.actions-row {
		display: flex;
		gap: 1rem;
		justify-content: center;
		width: 100%;
	}

	.send-btn {
		min-width: 120px;
		background: var(--success);
	}

	.send-btn:hover:not(:disabled) {
		background: var(--success);
		filter: brightness(1.3);
	}

	.shuffle-btn {
		width: 3rem;
		height: 3rem;
		padding: 0.3rem;
		min-width: auto;
		flex-shrink: 0;
	}

	.die-icon {
		width: 100%;
		height: 100%;
		display: block;
		flex-shrink: 0;
		transition: transform 0.3s ease;
	}

	.shuffle-btn:hover .die-icon {
		transform: rotate(30deg) scale(1.1);
	}

	.shuffle-btn:active .die-icon {
		transform: rotate(-15deg) scale(0.95);
	}

	.send-btn {
		box-shadow: 0 4px 15px color-mix(in oklab, var(--success), transparent 50%);
	}

	@media (max-width: 768px) {
		.first-row {
			gap: 0.75rem;
		}
		.info-value { font-size: 1.6rem; }
		.composing-word { font-size: 2.2rem; }
		.letters-row { 
			grid-template-columns: repeat(5, 1fr); 
			max-width: 100%;
			gap: 0.4rem;
		}
		.letter-tile { 
			font-size: 1.25rem; 
		}
		.actions-row {
			gap: 0.5rem;
		}
	}

	@media (max-width: 480px) {
		.undo-btn, .send-btn {
			min-width: 90px;
			font-size: 1rem;
			padding: 0 0.75rem;
		}
		.shuffle-btn {
			width: 2.75rem;
			height: 2.75rem;
		}
		.composing-word {
			font-size: 1.8rem;
			letter-spacing: 0.15rem;
		}
	}
</style>