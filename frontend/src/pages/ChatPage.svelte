<script>
  // MIVA-CHAT: rendering only; App owns session state and side effects.
  import {
    Sprout,
    MessagesSquare,
    ArrowUpRight,
    Volume2,
    Send,
    Mic,
    Square,
    LoaderCircle,
  } from '@lucide/svelte';
  export let messages;
  export let busy;
  export let chatText;
  export let chatEnd;
  export let recording;
  export let recordSeconds;
  export let recognitionAvailable;
  export let dictationLang;
  export let recognizer;
  export let send;
  export let speak;
  export let startDictation;
</script>

<div class="heading">
  <div>
    <span class="eyebrow">A SAFE PLACE TO TRY</span>
    <h1>
      Ngobrol saja.
      <br />
      <em>Salah juga tak apa.</em>
    </h1>
    <p>Latihan bersama Miva, tutor AI-mu. Mulai dalam bahasa Inggris atau Indonesia.</p>
  </div>
</div>
<section class="panel chat-panel">
  <div class="chat-header">
    <span class="ellie-avatar"><Sprout size={25} /></span>
    <div>
      <h2>Miva, your conversation buddy</h2>
      <small>English & Bahasa Indonesia</small>
    </div>
    <span class="tag">ONE CONVERSATION AT A TIME</span>
  </div>
  <div class="chat-history" aria-live="polite">
    {#if !messages.length}
      <div class="chat-empty">
        <MessagesSquare size={38} />
        <h3>Let’s start with hello.</h3>
        <p>Ceritakan harimu atau coba situasi yang ingin kamu kuasai.</p>
        <div class="starter-prompts">
          {#each ['Let’s practice ordering a coffee.', 'Bantu saya memperkenalkan diri.', 'Ask me about my day.'] as prompt}
            <button disabled={!!busy} on:click={() => send(prompt)}>
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

    {#if busy === 'chat'}
      <div class="thinking">
        <LoaderCircle size={18} class="spin" /> Miva sedang menyiapkan balasan…
      </div>
    {/if}

    <div bind:this={chatEnd}></div>
  </div>
  <div class="chat-compose">
    <form on:submit|preventDefault={() => send()}>
      <textarea
        bind:value={chatText}
        maxlength="2000"
        rows="2"
        aria-label="Pesan untuk Miva"
        placeholder="Type a little English… atau mulai dalam bahasa Indonesia."
        on:keydown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            send();
          }
        }}
      ></textarea>
      <div class="compose-actions">
        <div class="dictation">
          <select bind:value={dictationLang} aria-label="Bahasa ucapan" disabled={recording}>
            <option value="en-US">English</option>
            <option value="id-ID">Indonesia</option>
          </select>

          {#if recording}
            <button type="button" class="record-btn recording" on:click={() => recognizer?.stop()}>
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
        </div>
        <button class="primary" disabled={!chatText.trim() || !!busy || recording}>
          Kirim <Send size={18} />
        </button>
      </div>
    </form>
    <p class="caption">
      Periksa hasil dikte sebelum dikirim. Ketersediaan mikrofon mengikuti dukungan browser.
    </p>
  </div>
</section>
