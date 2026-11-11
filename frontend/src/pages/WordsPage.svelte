<script>
  // MIVA-WORDS: rendering only; App owns session state and side effects.
  import { Library, LoaderCircle, Search, Plus } from '@lucide/svelte';
  import WordCard from '../components/WordCard.svelte';
  export let words;
  export let filteredWords;
  export let query;
  export let verbsOnly;
  export let busy;
  export let lookup;
  export let audioBusy;
  export let speechAvailable;
  export let playWord;
</script>

<div class="heading">
  <div>
    <span class="eyebrow">A WORLD OF WORDS</span>
    <h1>
      Kamus kecilmu,
      <br />
      <em>kemungkinan besar.</em>
    </h1>
    <p>Cari kata Inggris atau Indonesia, lalu kenali artinya lewat berbagai contoh.</p>
  </div>
  <span class="count-badge">{words.length} kata</span>
</div>
<section class="panel search-panel">
  <form class="search-form" on:submit|preventDefault={lookup}>
    <div class="search-input">
      <Search size={21} />
      <input
        bind:value={query}
        placeholder="Misalnya: speak, went, atau belajar"
        aria-label="Cari kosakata"
        maxlength="80"
      />
    </div>
    <button class="primary" disabled={!query.trim() || !!busy}>
      {#if busy === 'lookup'}
        <LoaderCircle class="spin" size={19} />
      {:else}
        Cari & tambah kata <Plus size={18} />
      {/if}
    </button>
  </form>
  <div class="search-meta">
    <p>Kata yang kamu pelajari tersimpan di sini.</p>
    <button
      class="filter-btn"
      class:active={verbsOnly}
      aria-pressed={verbsOnly}
      on:click={() => (verbsOnly = !verbsOnly)}
    >
      Hanya kata kerja
    </button>
  </div>
</section>
<div class="word-grid">
  {#each filteredWords as word}
    <WordCard
      {word}
      disabled={audioBusy || !speechAvailable}
      onPlay={(part, index) => playWord(word, part, index)}
    />
  {/each}
</div>

{#if !filteredWords.length}
  <div class="panel empty-state">
    <Library size={35} />
    <h2>Kata baru menunggu ditemukan.</h2>
    <p>Gunakan “Cari & tambah kata” untuk melengkapinya.</p>
  </div>
{/if}
