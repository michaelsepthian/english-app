<script>
  // MIVA-STUDY: rendering only; App owns session state and side effects.
  import {
    ArrowRight,
    ArrowUpRight,
    ChevronRight,
    Volume2,
    Play,
    Check,
    Headphones,
    LoaderCircle,
    CheckCircle2,
    Plus,
    History,
  } from '@lucide/svelte';
  import WordCard from '../components/WordCard.svelte';
  import QuestionCard from '../components/QuestionCard.svelte';
  import ReadingPractice from '../components/ReadingPractice.svelte';
  export let lesson;
  export let user;
  export let section;
  export let showHistory;
  export let historyItems;
  export let activeSections;
  export let answers;
  export let answered;
  export let audioBusy;
  export let speechAvailable;
  export let pronunciationAvailable;
  export let busy;
  export let draftError;
  export let chatText;
  export let openPlanner;
  export let loadSession;
  export let resultFor;
  export let scorePercent;
  export let questionsFor;
  export let playAudio;
  export let playWord;
  export let speak;
  export let choose;
  export let saveAnswers;
  export let submit;
  export let review;
  export let navigate;
</script>

<div class="heading">
  <div>
    <span class="eyebrow">YOUR OWN LITTLE LEARNING JOURNEY</span>
    <h1>{lesson.attempt ? 'Look at your progress.' : 'Let’s take the next step.'}</h1>
    <p>{lesson.title} · {lesson.day}</p>
  </div>
  <div class="heading-actions">
    <button class="secondary" on:click={() => (showHistory = !showHistory)}>
      <History size={18} /> Riwayat
    </button>
    <button class="secondary" on:click={openPlanner}><Plus size={18} /> Sesi baru</button>
  </div>
</div>

{#if showHistory}
  <section class="panel history-list">
    <h2>Sesi belajarmu</h2>

    {#each historyItems as item}
      <button on:click={() => loadSession(item.id)}>
        <span>
          <strong>{item.title}</strong>
          <small>{item.day} · {item.total} soal</small>
        </span>
        <b>{item.completed ? `${item.percent}%` : 'Lanjutkan'}</b>
        <ChevronRight size={18} />
      </button>
    {/each}
  </section>
{/if}

<div class="session-strip">
  <span>
    <span class="tiny-dot"></span>
    {lesson.vocabulary.length} kata · {lesson.tenses.length} tenses · {lesson.readings.length} reading
    · {lesson.listenings.length} listening
  </span>
  <strong>
    {lesson.attempt
      ? `${scorePercent()}% skor akhir`
      : `${answered}/${lesson.questions.length} terjawab`}
  </strong>
</div>
<div class="tabs">
  {#each activeSections as tab}
    <button
      class:active={section === tab.id}
      aria-pressed={section === tab.id}
      on:click={() => (section = tab.id)}
    >
      <svelte:component this={tab.icon} size={18} />{tab.label}
    </button>
  {/each}
</div>

{#if section === 'vocabulary'}
  <div class="section-intro">
    <h2>Kata kecil, banyak kemungkinan.</h2>
    <p>Ketuk arti untuk melatih ingatan. Dengarkan setiap bentuk kata dan lihat contoh lainnya.</p>
  </div>
  <div class="word-grid">
    {#each lesson.vocabulary as word}
      <WordCard
        {word}
        disabled={audioBusy || !speechAvailable}
        onPlay={(part, index) => playWord(word, part, index)}
      />
    {/each}
  </div>
{:else if section === 'tenses'}
  {#each lesson.tenses as tense, i}
    <section class="panel lesson-prose enter">
      <span class="tag">TENSE {i + 1} · {tense.name}</span>
      <h2>{tense.name}</h2>
      <p class="explanation">{tense.explanation}</p>

      {#if tense.uses?.length}
        <h3>Kapan digunakan?</h3>
        <ul class="lesson-list">
          {#each tense.uses as use}
            <li>{use}</li>
          {/each}
        </ul>
      {/if}

      <h3>Pola kalimat</h3>
      <div class="formula">{tense.formula}</div>

      {#if tense.rules?.length}
        <h3>Hal yang perlu diingat</h3>
        <ul class="lesson-list">
          {#each tense.rules as rule}
            <li>{rule}</li>
          {/each}
        </ul>
      {/if}

      <h3>Lihat lewat contoh</h3>
      <div class="tense-examples">
        {#each tense.examples as example, n}
          <div class="tense-example">
            <span>{String(n + 1).padStart(2, '0')}</span>
            <div>
              <p>{example.en}</p>
              <small>{example.id}</small>
            </div>
            <button
              class="icon"
              aria-label={`Dengarkan contoh ${n + 1}`}
              on:click={() => speak(example.en)}
            >
              <Volume2 size={19} />
            </button>
          </div>
        {/each}
      </div>

      {#if tense.commonMistakes?.length}
        <div class="common-mistakes">
          <h3>Kesalahan yang sering terjadi</h3>

          {#each tense.commonMistakes as mistake}
            <div>
              <p class="wrong-sentence">{mistake.en}</p>
              <p>{mistake.id}</p>
            </div>
          {/each}
        </div>
      {/if}

      <button
        class="text-btn"
        on:click={() => {
          navigate('chat');
          chatText = `Bantu saya memahami ${tense.name} dengan contoh untuk target saya: ${user.goal}`;
        }}
      >
        Bahas lebih lanjut dengan tutor <ArrowUpRight size={18} />
      </button>
    </section>
  {/each}
{:else if section === 'reading'}
  {#each lesson.readings as item, i (lesson.id + item.id)}
    <section class="panel lesson-prose reading enter">
      <span class="tag">READING {i + 1} / {lesson.readings.length}</span>
      <h2>{item.title}</h2>
      <button class="text-btn" on:click={() => speak(item.passage)}>
        <Volume2 size={19} /> Dengarkan bacaan
      </button>
      <p class="passage">{item.passage}</p>
      <ReadingPractice {item} lessonId={lesson.id} {pronunciationAvailable} onPlay={speak} />
    </section>
  {/each}
{:else if section === 'listening'}
  {#each lesson.listenings as item, i}
    <section class="listening-unit enter">
      <div class="panel listening-card">
        <span class="tag">LISTENING {i + 1} / {lesson.listenings.length}</span>
        <span class="empty-icon"><Headphones size={33} /></span>
        <h2>{item.title}</h2>
        <p>Dengarkan ceritanya. Setelah itu, jawab pertanyaan tepat di bawah ini.</p>
        <button
          class="primary"
          disabled={audioBusy || !speechAvailable}
          on:click={() => playAudio('listening', lesson.id, item.id)}
        >
          {#if audioBusy}
            <LoaderCircle class="spin" size={19} /> Menyiapkan suara…
          {:else}
            <Play size={19} /> Dengarkan bahan {i + 1}
          {/if}
        </button>
        <p class="caption">Boleh diputar berulang kali. Pengaturan suara ada di dalam pemutar.</p>

        {#if lesson.attempt}
          <details class="transcript">
            <summary>Lihat transkrip</summary>
            <p>
              {lesson.attempt.listeningTranscripts?.[item.id] || lesson.attempt.listeningTranscript}
            </p>
          </details>
        {/if}
      </div>
      <div class="inline-questions">
        <h3>Seberapa jauh kamu menangkap ceritanya?</h3>

        {#each questionsFor(item.id) as q, n}
          <QuestionCard
            question={q}
            number={n + 1}
            value={answers[q.id]}
            result={resultFor(q.id)}
            completed={!!lesson.attempt}
            onChoose={(value) => choose(q.id, value)}
            onListen={() => playAudio('listening', lesson.id, item.id)}
            compact
          />
        {/each}
      </div>
    </section>
  {/each}
{:else if section === 'quiz'}
  {#if lesson.attempt}
    <section class="panel result-card enter" id="session-result">
      <div class="result-decoration" aria-hidden="true">
        {#each Array(10) as _, i}
          <i style={`--i:${i}`}></i>
        {/each}
      </div>
      <span class="eyebrow">A MOMENT TO CELEBRATE</span>
      <div class="score-ring" style={`--score:${scorePercent() * 3.6}deg`}>
        <div>
          <strong>
            {scorePercent()}
            <small>%</small>
          </strong>
          <span>skor akhir</span>
        </div>
      </div>
      <h2>{scorePercent() >= 80 ? 'Look how far you’ve come.' : 'Every try is a step forward.'}</h2>
      <p>
        {lesson.attempt.score} dari {lesson.attempt.total} jawaban benar · {Math.floor(
          lesson.attempt.seconds / 60
        )} menit belajar.
      </p>
      <div class="score-note">
        <CheckCircle2 size={19} /> Hasilmu sudah masuk ke progres mingguan.
      </div>

      {#if lesson.attempt.review}
        <div class="personal-review">
          <h3>Catatan untuk langkah berikutnya</h3>
          <p>{lesson.attempt.review.summary}</p>
          <div class="review-columns">
            <div>
              <h4>Yang sudah kamu kuasai</h4>
              <ul>
                {#each lesson.attempt.review.strengths as text}
                  <li>{text}</li>
                {/each}
              </ul>
            </div>
            <div>
              <h4>Yang bisa ditingkatkan</h4>
              <ul>
                {#each lesson.attempt.review.improvements as text}
                  <li>{text}</li>
                {/each}
              </ul>
            </div>
          </div>
          <h4>Coba berikutnya</h4>
          <ul>
            {#each lesson.attempt.review.nextSteps as text}
              <li>{text}</li>
            {/each}
          </ul>
        </div>
      {:else}
        <button class="secondary" disabled={!!busy} on:click={review}>
          {#if busy === 'review'}
            <LoaderCircle size={18} class="spin" />
          {/if}
          Lihat rangkuman pribadi
        </button>
      {/if}
    </section>
  {/if}

  <div class="quiz-intro">
    <div>
      <span class="eyebrow">BRING IT ALL TOGETHER</span>
      <h2>{lesson.attempt ? 'Yuk, pelajari pembahasannya.' : 'Sedikit latihan, semakin paham.'}</h2>
      <p>
        {lesson.attempt
          ? 'Lihat jawabanmu, perbaikan, dan alasan di balik setiap jawaban.'
          : 'Latihan ini mencakup semua materi yang kamu pilih di awal.'}
      </p>
    </div>

    {#if !lesson.attempt}
      <button class="secondary" on:click={saveAnswers}>Simpan & jeda</button>
    {/if}
  </div>

  {#each lesson.questions as q, i}
    <QuestionCard
      question={q}
      number={i + 1}
      value={answers[q.id]}
      result={resultFor(q.id)}
      completed={!!lesson.attempt}
      onChoose={(value) => choose(q.id, value)}
      onListen={() => playAudio('listening', lesson.id, q.sourceId || 'listening-1')}
    />
  {/each}

  {#if !lesson.attempt}
    <div class="submit-bar">
      <div>
        <strong>{answered} dari {lesson.questions.length} soal terjawab</strong>
        <span class="caption">
          {draftError
            ? 'Jawaban belum tersimpan. Coba Simpan & jeda.'
            : 'Progres disimpan otomatis.'}
        </span>
        <div class="progress-track">
          <div style:width={`${(answered / lesson.questions.length) * 100}%`}></div>
        </div>
      </div>
      <button
        class="primary"
        disabled={answered !== lesson.questions.length || !!busy}
        on:click={submit}
      >
        {#if busy === 'submit'}
          <LoaderCircle class="spin" size={19} /> Memeriksa jawaban…
        {:else}
          Selesai & lihat hasil <Check size={19} />
        {/if}
      </button>
    </div>
  {/if}
{/if}

<div class="lesson-bottom">
  <button class="text-btn" on:click={saveAnswers} disabled={!!lesson.attempt}>
    Simpan progres
  </button>

  {#if section !== 'quiz'}
    <button
      class="secondary"
      on:click={() => {
        section =
          activeSections[
            Math.min(
              activeSections.findIndex((s) => s.id === section) + 1,
              activeSections.length - 1
            )
          ].id;
        window.scrollTo({ top: 0, behavior: 'smooth' });
      }}
    >
      Lanjutkan <ArrowRight size={18} />
    </button>
  {/if}
</div>
