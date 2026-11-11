<script>
  import { Volume2, RotateCcw, ChevronDown } from '@lucide/svelte';
  export let word;
  export let onPlay = () => {};
  export let disabled = false;
  let revealed = true;
  $: examples = word.examples?.length
    ? word.examples
    : [{ en: word.example, id: word.translation }];
</script>

<article class="panel word-card enter">
  <div class="section-head">
    <span class="tag">{word.partOfSpeech || 'Vocabulary'}</span>
    <button
      class="icon"
      aria-label={`Dengarkan ${word.word}`}
      {disabled}
      on:click={() => onPlay('word', 0)}
    >
      <Volume2 size={20} />
    </button>
  </div>
  <h3>{word.word}</h3>

  {#if word.pronunciation}
    <p class="ipa">{word.pronunciation}</p>
  {/if}

  <button
    class="meaning-toggle"
    class:covered={!revealed}
    on:click={() => (revealed = !revealed)}
    aria-label="Tampilkan atau sembunyikan arti"
  >
    {revealed ? word.meaning : 'Apa arti kata ini? Ketuk untuk melihat.'}<RotateCcw size={15} />
  </button>

  {#if word.v1}
    <div class="verb-forms">
      {#each ['v1', 'v2', 'v3'] as part, i}
        <button
          {disabled}
          title={`Dengarkan Verb ${i + 1}: ${word[part]}`}
          on:click={() => onPlay(part, 0)}
        >
          <small>VERB {i + 1}</small>
          <strong>{word[part]}</strong>
          <Volume2 size={15} />
        </button>
      {/each}
    </div>
  {/if}

  <div class="example">
    <p>{examples[0]?.en}</p>
    <small>{examples[0]?.id}</small>
    <button class="text-btn" {disabled} on:click={() => onPlay('example', 0)}>
      <Volume2 size={16} /> Dengarkan contoh
    </button>
  </div>

  {#if examples.length > 1}
    <details class="more-examples">
      <summary>{examples.length - 1} contoh lainnya <ChevronDown size={16} /></summary>

      {#each examples.slice(1) as e, i}
        <div class="example">
          <p>{e.en}</p>
          <small>{e.id}</small>
          <button class="text-btn" {disabled} on:click={() => onPlay('example', i + 1)}>
            <Volume2 size={16} /> Dengarkan
          </button>
        </div>
      {/each}
    </details>
  {/if}
</article>
