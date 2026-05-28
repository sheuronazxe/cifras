class SoundManager {
    private ctx: AudioContext | null = null;
    private boundCleanup = this.cleanup.bind(this);
    enabled = $state(typeof window !== 'undefined' ? localStorage.getItem('cifras_sound_enabled') !== 'false' : true);

    private getContext(): AudioContext | null {
        if (typeof window === 'undefined') return null;
        if (!this.ctx) {
            const AudioCtx = window.AudioContext || (window as any).webkitAudioContext;
            if (AudioCtx) {
                this.ctx = new AudioCtx();
                if (typeof window !== 'undefined') {
                    window.addEventListener('beforeunload', this.boundCleanup);
                }
            }
        }
        if (this.ctx && this.ctx.state === 'suspended') {
            this.ctx.resume();
        }
        return this.ctx;
    }

    private cleanup() {
        if (this.ctx) {
            this.ctx.close();
            this.ctx = null;
        }
        if (typeof window !== 'undefined') {
            window.removeEventListener('beforeunload', this.boundCleanup);
        }
    }

    toggle(state?: boolean) {
        this.enabled = state !== undefined ? state : !this.enabled;
        if (typeof window !== 'undefined') {
            localStorage.setItem('cifras_sound_enabled', String(this.enabled));
        }
    }

    private ensureContext(): AudioContext | null {
        if (!this.enabled) return null;
        return this.getContext();
    }

    private makeOscGain(ctx: AudioContext, type: OscillatorType = 'sine'): { osc: OscillatorNode, gain: GainNode } {
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.connect(gain);
        gain.connect(ctx.destination);
        osc.type = type;
        return { osc, gain };
    }

    playClick() {
        const ctx = this.ensureContext();
        if (!ctx) return;
        const { osc, gain } = this.makeOscGain(ctx, 'sine');
        // Quick, subtle high-to-low transient for a clean key click
        osc.frequency.setValueAtTime(1200, ctx.currentTime);
        osc.frequency.exponentialRampToValueAtTime(300, ctx.currentTime + 0.015);
        gain.gain.setValueAtTime(0.015, ctx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + 0.015);
        osc.start(ctx.currentTime);
        osc.stop(ctx.currentTime + 0.02);
    }

    playSuccess() {
        const ctx = this.ensureContext();
        if (!ctx) return;
        const notes = [523.25, 659.25, 783.99, 1046.50]; // C5, E5, G5, C6
        const duration = 0.08;
        notes.forEach((freq, index) => {
            const { osc, gain } = this.makeOscGain(ctx, 'triangle');
            osc.frequency.setValueAtTime(freq, ctx.currentTime + index * duration);
            gain.gain.setValueAtTime(0.05, ctx.currentTime + index * duration);
            gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + index * duration + 0.12);
            osc.start(ctx.currentTime + index * duration);
            osc.stop(ctx.currentTime + index * duration + 0.15);
        });
    }

    playError() {
        const ctx = this.ensureContext();
        if (!ctx) return;
        const { osc, gain } = this.makeOscGain(ctx, 'sawtooth');
        osc.frequency.setValueAtTime(180, ctx.currentTime);
        osc.frequency.linearRampToValueAtTime(100, ctx.currentTime + 0.25);
        gain.gain.setValueAtTime(0.08, ctx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.25);
        osc.start(ctx.currentTime);
        osc.stop(ctx.currentTime + 0.26);
    }

    playTick() {
        const ctx = this.ensureContext();
        if (!ctx) return;
        const { osc, gain } = this.makeOscGain(ctx, 'sine');
        osc.frequency.setValueAtTime(600, ctx.currentTime);
        gain.gain.setValueAtTime(0.04, ctx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.05);
        osc.start(ctx.currentTime);
        osc.stop(ctx.currentTime + 0.06);
    }

    playVictory() {
        const ctx = this.ensureContext();
        if (!ctx) return;
        const notes = [261.63, 329.63, 392.00, 523.25]; // C4, E4, G4, C5
        const time = ctx.currentTime;
        notes.forEach((freq, index) => {
            const { osc, gain } = this.makeOscGain(ctx, 'sine');
            osc.frequency.setValueAtTime(freq, time + index * 0.12);
            gain.gain.setValueAtTime(0.06, time + index * 0.12);
            gain.gain.exponentialRampToValueAtTime(0.001, time + index * 0.12 + 0.4);
            osc.start(time + index * 0.12);
            osc.stop(time + index * 0.12 + 0.45);
        });
        setTimeout(() => {
            const finalTime = ctx.currentTime;
            notes.forEach(freq => {
                const { osc, gain } = this.makeOscGain(ctx, 'triangle');
                osc.frequency.setValueAtTime(freq * 2, finalTime);
                gain.gain.setValueAtTime(0.04, finalTime);
                gain.gain.exponentialRampToValueAtTime(0.001, finalTime + 0.8);
                osc.start(finalTime);
                osc.stop(finalTime + 0.85);
            });
        }, 480);
    }
}

export const sound = new SoundManager();
