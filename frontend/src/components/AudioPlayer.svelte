<script>
  // MIVA-AUDIO: rendering only; App owns session state and side effects.
  import { Play, Pause, Headphones, Square, X, RotateCcw, SlidersHorizontal } from '@lucide/svelte';
  export let audioText;
  export let playerOpen;
  export let speaking;
  export let paused;
  export let voiceName;
  export let speechRate;
  export let voices;
  export let speak;
  export let stopSpeech;
  export let pauseSpeech;
</script>

<aside class="audio-dock enter" aria-label="Pemutar suara">
  <div class="player-title">
    <Headphones size={20} />
    <strong>Listen. Notice. Repeat.</strong>
    <button
      class="icon"
      aria-label="Tutup pemutar"
      on:click={() => {
        stopSpeech();
        playerOpen = false;
      }}
    >
      <X size={18} />
    </button>
  </div>
  <div class="speech-controls">
    <button class="secondary" on:click={() => speak(audioText)}>
      <RotateCcw size={17} /> Ulangi
    </button>
    <button class="secondary" disabled={!speaking} on:click={pauseSpeech}>
      {#if paused}
        <Play size={17} /> Lanjut
      {:else}
        <Pause size={17} /> Jeda
      {/if}
    </button>
    <button class="icon" disabled={!speaking} aria-label="Hentikan suara" on:click={stopSpeech}>
      <Square size={18} />
    </button>
  </div>
  <details class="audio-settings">
    <summary><SlidersHorizontal size={16} /> Pengaturan suara</summary>
    <label for="speech-voice">Suara Inggris</label>
    <select id="speech-voice" bind:value={voiceName}>
      <option value="">Otomatis</option>

      {#each voices as voice}
        <option value={voice.voiceURI}>{voice.name}</option>
      {/each}
    </select>
    <label for="speech-rate">Kecepatan · {Number(speechRate).toFixed(2)}×</label>
    <input id="speech-rate" type="range" bind:value={speechRate} min="0.6" max="1.2" step="0.05" />
  </details>
</aside>
