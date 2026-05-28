<script lang="ts">
	import { game } from "$lib/store.svelte";
	import { sound } from "$lib/sound.svelte";
	import Postit from "./Postit.svelte";

	interface NumberItem {
		id: number;
		val: number;
		used: boolean;
	}

	interface HistoryEntry {
		expr: string;
		availableNumbers: NumberItem[];
		bestDistance: number;
	}

	let firstNumber = $state<NumberItem | null>(null);
	let secondNumber = $state<NumberItem | null>(null);
	let selectedOp = $state<string | null>(null);
	let availableNumbers = $state<NumberItem[]>([]);
	let currentExpr = $state("");
	let pendingExpr = $state("");
	let bestDistance = $state(9999);
	let historyStack = $state<HistoryEntry[]>([]);
	let initialNumbers = $state<number[]>([]);
	let previousRoundKey = $state("");

	function resetSelection() {
		firstNumber = null;
		secondNumber = null;
		selectedOp = null;
		pendingExpr = "";
	}

	function resetBoard(numbers: number[]) {
		availableNumbers = numbers.map((val) => ({
			id: game.nextId(),
			val,
			used: false,
		}));
		currentExpr = "";
		pendingExpr = "";
		bestDistance = 9999;
		historyStack = [];
		resetSelection();
	}

	$effect(() => {
		const state = game.gameState.state;
		const roundType = game.gameState.roundType;
		const numbers = game.gameState.numbers;
		const roundKey = game.roundKey;

		if (
			state === "PLAYING" &&
			roundType === "CIFRAS" &&
			numbers?.length > 0
		) {
			if (roundKey !== previousRoundKey) {
				initialNumbers = [...numbers];
				resetBoard(numbers);
				previousRoundKey = roundKey;
			}
		}
	});

	function clickNumber(num: NumberItem) {
		if (num.used) return;
		sound.playClick();
		if (firstNumber?.id === num.id) {
			if (secondNumber) {
				firstNumber = secondNumber;
				secondNumber = null;
			} else {
				firstNumber = null;
			}
			updatePendingExpr();
			checkAndExecute();
			return;
		}
		if (secondNumber?.id === num.id) {
			secondNumber = null;
			updatePendingExpr();
			checkAndExecute();
			return;
		}
		if (!firstNumber) firstNumber = num;
		else if (!secondNumber) secondNumber = num;
		else {
			firstNumber = secondNumber;
			secondNumber = num;
		}
		updatePendingExpr();
		checkAndExecute();
	}

	function clickOp(op: string) {
		sound.playClick();
		if (selectedOp === op) selectedOp = null;
		else selectedOp = op;
		updatePendingExpr();
		checkAndExecute();
	}

	function updatePendingExpr() {
		let parts: (number | string)[] = [];
		if (firstNumber) parts.push(firstNumber.val);
		if (selectedOp) parts.push(selectedOp);
		if (secondNumber) parts.push(secondNumber.val);
		pendingExpr = parts.join(" ");
	}

	let executing = $state(false);

	function checkAndExecute() {
		if (firstNumber && secondNumber && selectedOp && !executing) {
			executing = true;
			setTimeout(() => {
				executeOp();
				executing = false;
			}, 200);
		}
	}

	function executeOp() {
		if (!firstNumber || !secondNumber || !selectedOp) return;
		const a = firstNumber.val;
		const b = secondNumber.val;
		let res = 0;
		let valid = true;

		if (selectedOp === "+") res = a + b;
		else if (selectedOp === "-") {
			res = a - b;
			if (res <= 0) valid = false;
		} else if (selectedOp === "×") res = a * b;
		else if (selectedOp === "÷") {
			if (b === 0 || a % b !== 0) valid = false;
			else res = a / b;
		}

		if (!valid) {
			game.addToast(
				"Operación inválida (resultado debe ser positivo y división exacta)",
				"error",
			);
			resetSelection();
			return;
		}

		historyStack = [
			...historyStack,
			{
				expr: currentExpr,
				availableNumbers: $state.snapshot(availableNumbers),
				bestDistance,
			},
		];

		const firstId = firstNumber.id;
		const secondId = secondNumber.id;
		const op = selectedOp;
		resetSelection();

		availableNumbers = availableNumbers.filter(
			(n) => n.id !== firstId && n.id !== secondId,
		);
		const newLine = `${a} ${op} ${b} = ${res}`;
		currentExpr = currentExpr ? `${currentExpr}\n${newLine}` : newLine;

		if (availableNumbers.length > 0) {
			const newId = game.nextId();
			availableNumbers = [
				...availableNumbers,
				{ id: newId, val: res, used: false },
			];
		}

		const target = game.gameState.targetNumber;
		const diff = Math.abs(target - res);
		if (diff < bestDistance) {
			bestDistance = diff;
			const normalizedExpr = currentExpr
				.replace(/×/g, "*")
				.replace(/÷/g, "/");
			game.submitCifras(res, normalizedExpr);
		}
	}

	function undoLast() {
		if (historyStack.length === 0) return;
		sound.playClick();
		const prev = historyStack[historyStack.length - 1];
		if (!prev) return;
		historyStack = historyStack.slice(0, -1);
		currentExpr = prev.expr;
		availableNumbers = prev.availableNumbers;
		bestDistance = prev.bestDistance;
		resetSelection();
	}

	function clearAll() {
		sound.playClick();
		resetBoard(initialNumbers);
	}
</script>

<div class="game-layout cifras-view">
	<div class="first-row">
		<div class="info-panel">
			<div class="info-label">OBJETIVO</div>
			<div class="info-value">{game.gameState.targetNumber}</div>
		</div>
		<div class="postit-container">
			<Postit
				fixed={true}
				content={currentExpr}
				pending={pendingExpr}
				onClear={clearAll}
			/>
		</div>
		<div class="info-panel">
			<div class="info-label">TIEMPO</div>
			<div class="info-value" class:urgent={game.timeRemaining <= 10}>{game.timeRemaining}</div>
		</div>
	</div>

	<div class="second-row numbers">
		{#each availableNumbers as num (num.id)}
			<button
				class="num-btn game-tile"
				class:selected={firstNumber?.id === num.id ||
					secondNumber?.id === num.id}
				aria-label="Seleccionar número {num.val}"
				onclick={() => clickNumber(num)}
			>
				{num.val}
			</button>
		{/each}
	</div>

	<div class="third-row ops">
		{#each ["+", "-", "×", "÷"] as op}
			<button
				class="action-btn"
				class:selected={selectedOp === op}
				aria-label="Operación {op === '+' ? 'suma' : op === '-' ? 'resta' : op === '×' ? 'multiplicación' : 'división'}"
				onclick={() => clickOp(op)}
			>
				{op}
			</button>
		{/each}
		<button
			class="action-btn undo-btn"
			disabled={historyStack.length === 0}
			onclick={undoLast}
		>
			Atrás
		</button>
	</div>
</div>

<style>
	.first-row {
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		align-items: center;

		width: 100%;
	}

	.second-row,
	.third-row {
		display: flex;
		gap: 1rem;
		justify-content: center;
		flex-wrap: wrap;
	}

	.num-btn {
		width: 80px;
		height: 60px;
		font-size: 1.5rem;
		font-weight: bold;
		color: var(--text);
	}

	.num-btn.selected {
		box-shadow: 0 0 0 2px var(--accent);
		background: var(--bg);
		border-color: var(--accent);
	}

	.action-btn {
		min-width: 60px;
	}

	.action-btn:not(.undo-btn) {
		padding: 0;
	}

	.postit-container {
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.action-btn.selected {
		background: var(--accent);
		color: var(--text);
		box-shadow: 0 0 0 2px var(--accent-muted);
	}

	@media (max-width: 768px) {
		.first-row {
			grid-template-columns: 1fr 1fr;
			gap: 1rem;
		}
		.postit-container {
			order: 3;
			grid-column: span 2;
			display: flex;
			justify-content: center;
			margin-top: 0.625rem;
		}

		.num-btn {
			min-width: 60px;
			font-size: 1.25rem;
			height: 50px;
		}
	}
</style>