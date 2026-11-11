<script>
  import { Check, Headphones, CheckCircle2, Lightbulb } from '@lucide/svelte';
  import { skills } from '../api';
  export let question;
  export let number = 1;
  export let value;
  export let result = null;
  export let completed = false;
  export let onChoose = () => {};
  export let onListen = () => {};
  export let compact = false;
</script>

<article class="panel question enter" class:compact>
  <div class="question-number">
    <span>{String(number).padStart(2, '0')}</span>
    <small>{skills[question.skill]}</small>

    {#if result}
      <strong class:incorrect={!result.correct}>
        {result.correct ? 'Benar' : 'Perlu diulang'}
      </strong>
    {/if}
  </div>
  <h3>{question.prompt}</h3>

  {#if question.skill === 'listening'}
    <button class="text-btn" on:click={onListen}>
      <Headphones size={17} /> Dengarkan kembali
    </button>
  {/if}

  <div class="options">
    {#each question.options as option, index}
      <button
        class:selected={value === index && !result}
        class:correct={result?.answer === index}
        class:wrong={result && !result.correct && result.chosen === index}
        disabled={completed}
        aria-pressed={result ? result.chosen === index : value === index}
        on:click={() => onChoose(index)}
      >
        <span>{String.fromCharCode(65 + index)}</span>
        <p>{option}</p>

        {#if result?.answer === index}
          <Check size={18} />
        {/if}
      </button>
    {/each}
  </div>

  {#if result}
    <div class="feedback">
      <strong>
        <Lightbulb size={18} />
        {result.correct ? 'Tepat! Ini alasannya.' : 'Mari lihat perbaikannya.'}
      </strong>

      {#if !result.correct}
        <p>
          Jawabanmu: <b>{question.options[result.chosen]}</b>
        </p>
        <p>
          Jawaban yang benar: <b>{question.options[result.answer]}</b>
        </p>
      {/if}

      <p>{result.explanation}</p>
    </div>
  {/if}
</article>
