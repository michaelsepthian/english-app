<script>
  // MIVA-PLAN: rendering only; App owns session state and side effects.
  import {
    Sprout,
    BookOpen,
    Library,
    Target,
    ArrowRight,
    Check,
    Clock3,
    Headphones,
    LoaderCircle,
  } from '@lucide/svelte';
  import { tenses } from '../api';
  export let user;
  export let lesson;
  export let planner;
  export let requestId;
  export let job;
  export let minimum;
  export let estimatedMinutes;
  export let busy;
  export let navigate;
  export let togglePart;
  export let toggleTense;
  export let fixQuestions;
  export let prepare;
</script>

<div class="heading">
  <div>
    <span class="eyebrow">MAKE IT YOUR OWN</span>
    <h1>
      Hari ini mau
      <br />
      <em>belajar apa saja?</em>
    </h1>
    <p>Pilih materinya, tentukan porsinya. Kita siapkan sesi yang mengarah ke targetmu.</p>
  </div>

  {#if lesson}
    <button class="secondary" on:click={() => navigate('study')}>
      Kembali ke sesi <ArrowRight size={18} />
    </button>
  {/if}
</div>

{#if job && ['queued', 'running'].includes(job.status)}
  <section class="panel preparation">
    <div class="prep-orb">
      <Sprout size={43} />
      <span></span>
      <span></span>
    </div>
    <span class="eyebrow">SOMETHING GOOD IS GROWING</span>
    <h2>{job.stage || 'Menyiapkan sesi belajarmu'}</h2>
    <p>
      Semakin banyak materi yang kamu pilih, semakin lama persiapannya. Pilihan dan progres
      persiapanmu tersimpan.
    </p>
    <div class="progress-track"><div style:width={`${Math.max(5, job.progress)}%`}></div></div>
    <strong>{job.progress}%</strong>
    <div class="prep-summary">
      <span>{planner.vocabulary} kosakata</span>
      <span>{planner.tenses.length} tenses</span>
      <span>{planner.reading} bacaan</span>
      <span>{planner.listening} listening</span>
      <span>{planner.questions} soal</span>
    </div>
    <p class="caption">
      Kamu boleh membuka halaman lain. Sesi akan tersedia setelah selesai disiapkan.
    </p>
  </section>
{:else}
  <form class="planner-form" on:submit|preventDefault={prepare}>
    <div class="plan-goal">
      <Target size={24} />
      <div>
        <small>SESI INI AKAN MENGARAH KE TARGETMU</small>
        <p>{user.goal}</p>
      </div>
      <button class="text-btn" type="button" on:click={() => navigate('profile')}>Ubah</button>
    </div>
    <div class="plan-grid">
      <section class="panel plan-card" class:chosen={planner.vocabulary !== 0}>
        <button
          class="plan-toggle"
          type="button"
          aria-pressed={planner.vocabulary !== 0}
          on:click={() => togglePart('vocabulary')}
        >
          <span class="plan-icon"><Library size={25} /></span>
          <span>
            <strong>Kosakata</strong>
            <small>Kata baru, banyak konteks.</small>
          </span>
          <span class="check-box">
            {#if planner.vocabulary !== 0}
              <Check size={18} />
            {/if}
          </span>
        </button>

        {#if planner.vocabulary !== 0}
          <label for="word-count">Berapa kata ingin dipelajari?</label>
          <div class="count-input">
            <input
              id="word-count"
              type="number"
              bind:value={planner.vocabulary}
              min="1"
              max="30"
              required
              on:change={fixQuestions}
            />
            <span>kata · maksimal 30</span>
          </div>
          <p class="caption">
            Setiap kata disertai arti, bentuk verb jika ada, dan minimal 3 contoh kalimat.
          </p>
        {/if}
      </section>
      <section class="panel plan-card" class:chosen={planner.tenses.length > 0}>
        <button
          class="plan-toggle"
          type="button"
          aria-pressed={planner.tenses.length > 0}
          on:click={() => togglePart('tenses')}
        >
          <span class="plan-icon peach"><BookOpen size={25} /></span>
          <span>
            <strong>Tenses</strong>
            <small>Pahami pola, bukan hanya rumus.</small>
          </span>
          <span class="check-box">
            {#if planner.tenses.length}
              <Check size={18} />
            {/if}
          </span>
        </button>

        {#if planner.tenses.length}
          <p class="field-label">Pilih maksimal 3 tenses</p>
          <div class="tense-chips">
            {#each tenses as tense}
              <button
                type="button"
                class:active={planner.tenses.includes(tense)}
                aria-pressed={planner.tenses.includes(tense)}
                on:click={() => toggleTense(tense)}
              >
                {tense}
              </button>
            {/each}
          </div>
          <p class="caption">
            Penjelasan detail, rumus positif/negatif/tanya, dan 6 contoh setiap tense.
          </p>
        {/if}
      </section>
      <section class="panel plan-card" class:chosen={planner.reading !== 0}>
        <button
          class="plan-toggle"
          type="button"
          aria-pressed={planner.reading !== 0}
          on:click={() => togglePart('reading')}
        >
          <span class="plan-icon lavender"><BookOpen size={25} /></span>
          <span>
            <strong>Reading</strong>
            <small>Baca cerita, berani bersuara.</small>
          </span>
          <span class="check-box">
            {#if planner.reading !== 0}
              <Check size={18} />
            {/if}
          </span>
        </button>

        {#if planner.reading !== 0}
          <label for="reading-count">Berapa bacaan hari ini?</label>
          <div class="count-input">
            <input
              id="reading-count"
              type="number"
              bind:value={planner.reading}
              min="1"
              max="5"
              required
              on:change={fixQuestions}
            />
            <span>bacaan · maksimal 5</span>
          </div>
          <p class="caption">
            Cerita sesuai level, contoh suara, dan latihan membaca dengan lantang.
          </p>
        {/if}
      </section>
      <section class="panel plan-card" class:chosen={planner.listening !== 0}>
        <button
          class="plan-toggle"
          type="button"
          aria-pressed={planner.listening !== 0}
          on:click={() => togglePart('listening')}
        >
          <span class="plan-icon blue"><Headphones size={25} /></span>
          <span>
            <strong>Listening</strong>
            <small>Dengarkan, lalu pahami ceritanya.</small>
          </span>
          <span class="check-box">
            {#if planner.listening !== 0}
              <Check size={18} />
            {/if}
          </span>
        </button>

        {#if planner.listening !== 0}
          <label for="listening-count">Berapa bahan ingin didengarkan?</label>
          <div class="count-input">
            <input
              id="listening-count"
              type="number"
              bind:value={planner.listening}
              min="1"
              max="5"
              required
              on:change={fixQuestions}
            />
            <span>bahan · maksimal 5</span>
          </div>
          <p class="caption">Setiap bahan punya pertanyaan pemahaman langsung di bawah pemutar.</p>
        {/if}
      </section>
    </div>
    <section class="panel question-plan">
      <div>
        <span class="eyebrow">PUT IT INTO PRACTICE</span>
        <h2>Berapa soal latihan?</h2>
        <p>Soal mencakup semua bagian yang kamu pilih. Minimal {minimum || 1}, maksimal 30 soal.</p>
      </div>
      <div class="question-count">
        <input
          type="number"
          aria-label="Jumlah soal latihan"
          bind:value={planner.questions}
          min={minimum || 1}
          max="30"
          required
          on:change={fixQuestions}
        />
        <input
          type="range"
          aria-label="Atur jumlah soal"
          bind:value={planner.questions}
          min={minimum || 1}
          max="30"
          on:input={() => (requestId = '')}
        />
      </div>
    </section>
    <div class="planner-submit">
      <div>
        <Clock3 size={20} />
        <span>
          Estimasi <strong>{estimatedMinutes} menit</strong>
          <small>Belajar sesuai ritmemu. Bisa dijeda.</small>
        </span>
      </div>
      <button class="primary" disabled={!!busy}>
        {#if busy === 'prepare'}
          <LoaderCircle size={19} class="spin" /> Menyiapkan…
        {:else}
          Siapkan sesi saya <ArrowRight size={20} />
        {/if}
      </button>
    </div>
  </form>
{/if}
