import { sound } from './sound.svelte';

interface Toast {
    id: number;
    message: string;
    type: string;
}

type ServerMessageType = 'SYNC' | 'WELCOME' | 'WORD_ACCEPTED' | 'TOAST';

interface ServerMessage {
    type: ServerMessageType;
    payload: any;
}

interface Player {
    id: string;
    name: string;
    score: number;
    isReady: boolean;
}

interface GameState {
    state: string;
    currentRound: number;
    players: Player[];
    chooser: string;
    chooserId: string;
    targetNumber: number;
    numbers: number[];
    letters: string[];
    endTime: number;
    serverTime: number;
    winner: string;
    solution: string;
    solutionSteps: string[];
    exactSolutionSteps: string[];
    otherResults: PlayerResult[];
    roundType: string;
}

export interface PlayerResult {
    playerId?: string;
    name: string;
    finalNumber?: number;
    distance?: number;
    expression?: string;
    word?: string;
    points: number;
}

let toastCounter = 0;
let idCounter = 0;

export class GameStore {
    state = $state('IDLE');
    gameState = $state<GameState>({
        state: 'LOBBY',
        currentRound: 0,
        players: [],
        chooser: '',
        chooserId: '',
        targetNumber: 0,
        numbers: [],
        letters: [],
        endTime: 0,
        serverTime: 0,
        winner: '',
        solution: '',
        solutionSteps: [],
        exactSolutionSteps: [],
        otherResults: [],
        roundType: 'NONE'
    });
    
    playerName = $state(typeof window !== 'undefined' ? localStorage.getItem('cifras_player_name') || '' : '');
    myID = $state<string | null>(null);
    myBestWord = $state<string | null>(null);
    toasts = $state<Toast[]>([]);
    timeRemaining = $state(0);
    ws: WebSocket | null = null; // No es $state: WebSocket no debe ser reactivo
    reconnectAttempts = 0;
    maxReconnectAttempts = 5;
    static readonly COUNTDOWN_SECONDS = 30;

    get me() {
        return this.gameState.players.find(
            (p) => p.id === this.myID || (this.myID === null && p.name === this.playerName)
        );
    }

    get isMeReady() {
        return this.me?.isReady ?? false;
    }

    get isChooser() {
        return this.myID
            ? this.gameState.chooserId === this.myID
            : this.gameState.chooser === this.playerName;
    }

    get roundKey() {
        return `${this.gameState.state}-${this.gameState.roundType}-${this.gameState.currentRound}`;
    }

    isMe(player: Player) {
        return player.id === this.myID || (this.myID === null && player.name === this.playerName);
    }

    get progressPercent() {
        if (this.gameState.endTime <= 0) return 0;
        return Math.min(100, (1 - this.timeRemaining / GameStore.COUNTDOWN_SECONDS) * 100);
    }

    connect(name: string) {
        this.playerName = name;
        this.state = 'CONNECTING';
        this.reconnectAttempts = 0;
        this.establishWs();
    }

    reconnect() {
        if (this.state !== 'DISCONNECTED' && this.state !== 'IDLE') return;
        this.state = 'CONNECTING';
        this.reconnectAttempts = 0;
        this.establishWs();
    }

    establishWs() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = import.meta.env.DEV ? `${window.location.hostname}:8080` : window.location.host;
        const idParam = this.myID ? `?id=${encodeURIComponent(this.myID)}` : '';
        const ws = new WebSocket(`${protocol}//${host}/ws${idParam}`);
        this.ws = ws;

        ws.onopen = () => {
            this.state = 'CONNECTED';
            this.reconnectAttempts = 0;
            this.send('NAME', { name: this.playerName });
        };

        ws.onmessage = (event) => {
            let data: ServerMessage;
            try {
                data = JSON.parse(event.data);
            } catch (e) {
                console.error('Failed to parse WebSocket message:', e);
                return;
            }
            if (data.type === 'SYNC') {
                const oldState = this.gameState.state;
                this.gameState = data.payload;
                if (oldState === 'PLAYING' && this.gameState.state === 'FINISHED') {
                    if (this.gameState.winner && this.gameState.winner !== 'Nadie') {
                        sound.playVictory();
                    } else {
                        sound.playError();
                    }
                }
            } else if (data.type === 'WELCOME') {
                this.myID = data.payload.id;
                this.playerName = data.payload.name;
                if (typeof window !== 'undefined') {
                    localStorage.setItem('cifras_player_name', this.playerName);
                }
            } else if (data.type === 'WORD_ACCEPTED') {
                this.myBestWord = data.payload.word;
            } else if (data.type === 'TOAST') {
                this.addToast(data.payload.message, data.payload.type);
                if (data.payload.type === 'success') {
                    sound.playSuccess();
                } else if (data.payload.type === 'error') {
                    sound.playError();
                }
            }
        };

        ws.onerror = (event) => {
            console.error('WebSocket error:', event);
        };

        ws.onclose = () => {
            if (this.ws !== ws) return;
            if (this.state === 'DISCONNECTED' || this.state === 'IDLE') return;

            this.state = 'RECONNECTING';
            if (this.reconnectAttempts < this.maxReconnectAttempts) {
                const backoff = Math.pow(2, this.reconnectAttempts) * 1000 + Math.random() * 500;
                setTimeout(() => this.establishWs(), backoff);
                this.reconnectAttempts++;
            } else {
                this.state = 'DISCONNECTED';
                this.addToast("Conexión perdida. Por favor, recarga la página.", "error");
            }
        };
    }

    disconnect() {
        if (this.ws) {
            const ws = this.ws;
            this.ws = null;
            ws.close();
        }
        this.state = 'IDLE';
    }

    send(type: string, payload: Record<string, unknown> = {}) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type, ...payload }));
        }
    }

    ready() {
        this.send('READY');
    }

    chooseVowels(count: number) {
        this.send('CHOOSE_VOWELS', { vowels: count });
    }

    submitCifras(number: number, expr: string) {
        this.send('SUBMIT', { number, expr });
    }

    submitLetras(word: string) {
        this.send('SUBMIT', { word });
    }

    addToast(message: string, type = 'info') {
        const id = ++toastCounter;
        this.toasts.push({ id, message, type });
        setTimeout(() => {
            this.toasts = this.toasts.filter(t => t.id !== id);
        }, 3000);
    }

    nextId() {
        return ++idCounter;
    }
}

export const game = new GameStore();
