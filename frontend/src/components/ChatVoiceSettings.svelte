<script>
  import { onMount, tick } from 'svelte';
  import { Settings2, X, RotateCcw, Play, Pause, Square } from '@lucide/svelte';

  export let dictationLang;
  export let recording;
  export let speechAvailable;
  export let audioText;
  export let speaking;
  export let paused;
  export let voiceName;
  export let speechRate;
  export let voices;
  export let speak;
  export let stopSpeech;
  export let pauseSpeech;

  let open = false,
    root,
    trigger,
    panel;
  let availableHeight = 400;
  $: if (recording) close(false);
  $: if (voiceName && !voices.some((voice) => voice.voiceURI === voiceName)) voiceName = '';

  function measure() {
    if (!root) return;
    const composer = root.closest('.chat-compose');
    const header = root.closest('.chat-panel')?.querySelector('.chat-header');
    const viewport = window.visualViewport;
    const top = Math.max(header?.getBoundingClientRect().bottom ?? 0, viewport?.offsetTop ?? 0);
    availableHeight = Math.max(80, composer.getBoundingClientRect().top - top - 16);
  }
  function close(restoreFocus = true) {
    open = false;
    if (restoreFocus) trigger?.focus({ preventScroll: true });
  }
  async function toggle() {
    if (open) return close();
    measure();
    open = true;
    await tick();
    panel?.focus({ preventScroll: true });
  }
  onMount(() => {
    const outside = (event) => {
      if (open && !root.contains(event.target) && !panel?.contains(event.target)) close(false);
    };
    const escape = (event) => {
      if (open && event.key === 'Escape') {
        event.preventDefault();
        close();
      }
    };
    const observer = new ResizeObserver(measure);
    observer.observe(root.closest('.chat-compose'));
    document.addEventListener('pointerdown', outside);
    document.addEventListener('keydown', escape);
    window.addEventListener('resize', measure);
    window.visualViewport?.addEventListener('resize', measure);
    return () => {
      observer.disconnect();
      document.removeEventListener('pointerdown', outside);
      document.removeEventListener('keydown', escape);
      window.removeEventListener('resize', measure);
      window.visualViewport?.removeEventListener('resize', measure);
    };
  });
</script>

<span class="chat-settings-trigger" bind:this={root}>
  <button
    type="button"
    class="record-btn"
    bind:this={trigger}
    aria-label="Pengaturan suara"
    aria-expanded={open}
    aria-controls="chat-voice-settings"
    aria-haspopup="dialog"
    on:click={toggle}
  >
    <Settings2 size={20} />
  </button>
</span>
{#if open}
  <div
    class="chat-voice-settings"
    id="chat-voice-settings"
    role="dialog"
    aria-label="Pengaturan suara"
    tabindex="-1"
    bind:this={panel}
    style:max-height={`${availableHeight}px`}
  >
    <div class="chat-settings-heading">
      <h3>Pengaturan suara</h3>
      <button
        type="button"
        class="icon"
        aria-label="Tutup pengaturan suara"
        on:click={() => close()}
      >
        <X size={18} />
      </button>
    </div>
    <fieldset class="chat-language-options" disabled={recording}>
      <legend>Bahasa dikte</legend>
      <div>
        <label>
          <input
            type="radio"
            name="chat-dictation-language"
            value="id-ID"
            bind:group={dictationLang}
          />
          Indonesia
        </label>
        <label>
          <input
            type="radio"
            name="chat-dictation-language"
            value="en-US"
            bind:group={dictationLang}
          />
          English
        </label>
      </div>
      <p>Bahasa yang dikenali saat memakai mikrofon.</p>
    </fieldset>
    <div class="chat-playback-settings">
      <h4>Pemutar suara Miva</h4>
      <div class="chat-playback-actions">
        <button
          type="button"
          class="secondary"
          disabled={!speechAvailable || !audioText}
          on:click={() => speak(audioText)}
        >
          <RotateCcw size={16} /> Ulangi
        </button>
        <button
          type="button"
          class="secondary"
          disabled={!speechAvailable || !speaking}
          on:click={pauseSpeech}
        >
          {#if paused}<Play size={16} /> Lanjut{:else}<Pause size={16} /> Jeda{/if}
        </button>
        <button
          type="button"
          class="icon"
          aria-label="Hentikan suara"
          disabled={!speechAvailable || !speaking}
          on:click={stopSpeech}
        >
          <Square size={17} />
        </button>
      </div>
      {#if !audioText}<p>Pilih “Dengarkan” pada balasan Miva untuk memutar suara.</p>{/if}
      {#if !speechAvailable}<p>Pembaca suara belum tersedia di browser ini.</p>{/if}
      <label for="chat-playback-voice">Suara playback Inggris</label>
      <select
        id="chat-playback-voice"
        bind:value={voiceName}
        disabled={!speechAvailable || !voices.length}
      >
        <option value="">{voices.length ? 'Otomatis' : 'Suara default perangkat'}</option>
        {#each voices as voice}<option value={voice.voiceURI}>{voice.name}</option>{/each}
      </select>
      <label for="chat-playback-rate">Kecepatan · {Number(speechRate).toFixed(2)}×</label>
      <input
        id="chat-playback-rate"
        type="range"
        bind:value={speechRate}
        min="0.6"
        max="1.2"
        step="0.05"
        disabled={!speechAvailable}
      />
    </div>
  </div>
{/if}
