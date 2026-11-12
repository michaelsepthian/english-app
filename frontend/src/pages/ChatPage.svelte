<script>
  // App owns conversation and speech state; this page owns viewport and scroll behavior.
  import { onMount, afterUpdate, tick } from 'svelte';
  import {
    Sprout,
    MessagesSquare,
    ArrowUpRight,
    Volume2,
    Send,
    Mic,
    Square,
    LoaderCircle,
    ArrowDown,
    AlertCircle,
    X,
  } from '@lucide/svelte';
  export let messages;
  export let pendingChat = '';
  export let failedChat = '';
  export let restoreFailedChat;
  export let busy;
  export let chatText;
  export let keyboardOpen = false;
  export let viewportHeight = '';
  export let viewportTop = 0;
  export let error = '';
  export let dismissError;
  let history, textarea;
  let nearBottom = true,
    unread = false,
    followNext = true;
  let previousMessages, previousBusy, previousText, previousPending;

  function scrollLatest(smooth = false) {
    if (!history) return;
    history.scrollTo({
      top: messages.length || busy === 'chat' ? history.scrollHeight : 0,
      behavior:
        smooth && !window.matchMedia('(prefers-reduced-motion: reduce)').matches
          ? 'smooth'
          : 'instant',
    });
    unread = false;
  }

  function trackScroll() {
    nearBottom = history.scrollHeight - history.scrollTop - history.clientHeight < 80;
    if (nearBottom) unread = false;
    else followNext = false;
  }

  function resizeInput() {
    if (!textarea) return;
    textarea.style.height = 'auto';
    textarea.style.height = `${Math.min(textarea.scrollHeight, 120)}px`;
  }

  function submit(text = chatText) {
    if (!text.trim() || busy || recording) return;
    followNext = true;
    scrollLatest();
    send(text);
  }

  afterUpdate(() => {
    if (chatText !== previousText) {
      resizeInput();
      previousText = chatText;
    }
    if (pendingChat && pendingChat !== previousPending) followNext = true;
    if (messages !== previousMessages || busy !== previousBusy || pendingChat !== previousPending) {
      if (followNext || nearBottom) scrollLatest();
      else if (messages !== previousMessages) unread = true;
      if (messages !== previousMessages) followNext = false;
      previousMessages = messages;
      previousBusy = busy;
      previousPending = pendingChat;
    }
  });

  onMount(() => {
    const viewport = window.visualViewport;
    const mobile = window.matchMedia('(max-width: 640px)');
    let fullHeight = window.innerHeight;
    let orientation = window.screen.orientation?.type;
    function updateViewport() {
      const nextOrientation = window.screen.orientation?.type;
      if (nextOrientation !== orientation) {
        fullHeight = window.innerHeight;
        orientation = nextOrientation;
      }
      const height = viewport?.height ?? window.innerHeight;
      const focused = document.activeElement === textarea;
      if (!focused && !keyboardOpen) fullHeight = window.innerHeight;
      fullHeight = Math.max(fullHeight, window.innerHeight);
      keyboardOpen = mobile.matches && fullHeight - height > 120;
      viewportHeight = mobile.matches ? `${height}px` : '';
      viewportTop = mobile.matches ? (viewport?.offsetTop ?? 0) : 0;
      if (nearBottom) tick().then(() => scrollLatest());
    }
    const observer = new ResizeObserver(() => {
      if (nearBottom) scrollLatest();
    });
    observer.observe(history);
    updateViewport();
    scrollLatest();
    viewport?.addEventListener('resize', updateViewport);
    viewport?.addEventListener('scroll', updateViewport);
    window.addEventListener('resize', updateViewport);
    document.addEventListener('focusin', updateViewport);
    document.addEventListener('focusout', updateViewport);
    mobile.addEventListener('change', updateViewport);
    return () => {
      observer.disconnect();
      viewport?.removeEventListener('resize', updateViewport);
      viewport?.removeEventListener('scroll', updateViewport);
      window.removeEventListener('resize', updateViewport);
      document.removeEventListener('focusin', updateViewport);
      document.removeEventListener('focusout', updateViewport);
      mobile.removeEventListener('change', updateViewport);
      keyboardOpen = false;
      viewportHeight = '';
      viewportTop = 0;
    };
  });
  export let recording;
  export let recordSeconds;
  export let recognitionAvailable;
  export let stopDictation;
  export let send;
  export let speak;
  export let startDictation;
</script>

<section class="panel chat-panel">
  <div class="chat-header">
    <span class="ellie-avatar"><Sprout size={25} /></span>
    <div>
      <h2>Miva</h2>
      <small>Tutor AI · English & Indonesia</small>
    </div>
    <span class="chat-header-note">Sedikit latihan, makin percaya diri.</span>
  </div>
  <div class="chat-thread">
    <!-- Keyboard users need a focusable scroll region to read earlier messages. -->
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <div
      class="chat-history"
      bind:this={history}
      on:scroll={trackScroll}
      role="log"
      aria-label="Percakapan dengan Miva"
      aria-live="polite"
      tabindex="0"
    >
      {#if !messages.length && !pendingChat}
        <div class="chat-empty">
          <MessagesSquare size={38} />
          <h3>Ngobrol saja. Salah juga tak apa.</h3>
          <p>Ceritakan harimu atau coba situasi yang ingin kamu kuasai.</p>
          <div class="starter-prompts">
            {#each ['Let’s practice ordering a coffee.', 'Bantu saya memperkenalkan diri.', 'Ask me about my day.'] as prompt}
              <button disabled={!!busy || recording} on:click={() => submit(prompt)}>
                {prompt}<ArrowUpRight size={16} />
              </button>
            {/each}
          </div>
        </div>
      {/if}

      {#each messages as message}
        <div class="message" class:mine={message.role === 'user'}>
          <small>{message.role === 'user' ? 'KAMU' : 'MIVA · AI TUTOR'}</small>
          <p>{message.content}</p>

          {#if message.role === 'assistant'}
            <button class="text-btn" on:click={() => speak(message.content)}>
              <Volume2 size={17} /> Dengarkan
            </button>
          {/if}
        </div>
      {/each}

      {#if pendingChat}
        <div class="message mine" aria-label="Pesan sedang dikirim">
          <small>KAMU · MENGIRIM</small>
          <p>{pendingChat}</p>
        </div>
      {/if}
    </div>
    {#if unread}
      <button class="chat-latest" on:click={() => scrollLatest(true)}>
        <ArrowDown size={16} /> Pesan terbaru
      </button>
    {/if}
  </div>
  <div class="chat-compose">
    {#if error}
      <div class="chat-error" role="alert">
        <AlertCircle size={18} />
        <span>{error}</span>
        <button type="button" class="icon" aria-label="Tutup pesan" on:click={dismissError}>
          <X size={18} />
        </button>
      </div>
    {/if}
    {#if failedChat}
      <div class="chat-recovery">
        <p>Pesan belum terkirim:</p>
        <p class="failed-message">{failedChat}</p>
        <button
          type="button"
          class="text-btn"
          disabled={!!busy || recording || !!chatText.trim()}
          on:click={restoreFailedChat}
        >
          Pulihkan pesan
        </button>
        {#if chatText.trim()}<small>
            Kirim atau kosongkan draft saat ini untuk memulihkan pesan.
          </small>{/if}
      </div>
    {/if}
    {#if busy === 'chat'}
      <div class="chat-processing" role="status">
        <LoaderCircle size={18} class="spin" /> Miva sedang menyiapkan balasan…
      </div>
    {/if}
    <form on:submit|preventDefault={() => submit()}>
      <textarea
        bind:this={textarea}
        bind:value={chatText}
        maxlength="2000"
        readonly={recording}
        rows="1"
        aria-label="Pesan untuk Miva"
        placeholder="Tulis pesan dalam English atau Indonesia…"
        on:keydown={(e) => {
          if (
            e.key === 'Enter' &&
            !e.shiftKey &&
            !e.isComposing &&
            !window.matchMedia('(max-width: 640px)').matches
          ) {
            e.preventDefault();
            submit();
          }
        }}
      ></textarea>
      <div class="compose-actions">
        <div class="dictation">
          {#if recording}
            <button
              type="button"
              aria-label="Hentikan dikte"
              class="record-btn recording"
              on:click={stopDictation}
            >
              <Square size={17} />
              {recordSeconds}s
            </button>
          {:else}
            <button
              type="button"
              class="record-btn"
              disabled={!!busy || !recognitionAvailable}
              on:click={startDictation}
              aria-label="Dikte pesan"
            >
              <Mic size={20} />
            </button>
          {/if}
          <slot name="settings" />
        </div>
        <button class="primary" disabled={!chatText.trim() || !!busy || recording}>
          Kirim <Send size={18} />
        </button>
      </div>
    </form>
    <p class="caption">
      {recording
        ? `Mendengarkan · ${recordSeconds} detik. Pesan akan langsung dikirim setelah selesai bicara.`
        : recognitionAvailable
          ? 'Selesai bicara, pesan langsung dikirim.'
          : 'Dikte belum tersedia di browser ini. Kamu tetap bisa mengetik.'}
    </p>
  </div>
</section>
