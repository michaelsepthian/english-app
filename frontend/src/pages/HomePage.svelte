<script>
  // MIVA-PROGRESS: rendering only; App owns session state and side effects.
  import {
    BookOpen,
    Target,
    ArrowRight,
    ArrowUpRight,
    ChevronRight,
    Check,
    Clock3,
    CalendarDays,
    Leaf,
  } from '@lucide/svelte';
  import { skills } from '../api';
  export let user;
  export let stats;
  export let lesson;
  export let weekDays;
  export let greeting;
  export let navigate;
  export let openPlanner;
</script>

<div class="heading">
  <div>
    <span class="eyebrow">A LITTLE ENGLISH, A LITTLE MORE YOU</span>
    <h1>
      Hai, {greeting}.
      <br />
      <em>Siap tumbuh hari ini?</em>
    </h1>
    <p>Mulai dari yang kamu butuhkan. Kita belajar selangkah demi selangkah.</p>
  </div>
  <div class="date-card">
    <CalendarDays size={21} />{new Intl.DateTimeFormat('id-ID', {
      dateStyle: 'long',
      timeZone: 'Asia/Jakarta',
    }).format(new Date())}
  </div>
</div>
<section class="hero">
  <div class="hero-copy">
    <span class="hero-label">
      <span></span>
      YOUR NEXT SMALL STEP
    </span>
    <h2>
      Belajar yang pas,
      <br />
      untuk harimu.
    </h2>
    <p>
      Pilih kata, tenses, bacaan, dan cerita yang ingin kamu jelajahi. Satu sesi, sesuai
      kebutuhanmu.
    </p>
    <div class="hero-actions">
      <button class="light-btn" on:click={openPlanner}>
        Atur belajar hari ini <ArrowRight size={20} />
      </button>

      {#if lesson}
        <button class="hero-resume" on:click={() => navigate('study')}>
          {lesson.attempt ? 'Lihat sesi terakhir' : 'Lanjut sesi terakhir'}
          <ChevronRight size={18} />
        </button>
      {/if}
    </div>
    <small><Clock3 size={16} /> Ritme belajarmu, pilihanmu sendiri.</small>
  </div>
  <div class="miva-art" aria-hidden="true">
    <div class="art-ring ring-one"></div>
    <div class="art-ring ring-two"></div>
    <div class="art-card card-hello">
      hello
      <span>✳</span>
      <small>one word at a time.</small>
    </div>
    <div class="art-card card-grow">
      <span>grow a little</span>
      <b>every day.</b>
      <i>↗</i>
    </div>
    <div class="art-spark">✳</div>
    <span class="art-orb"></span>
  </div>
</section>
<div class="metrics">
  <div>
    <span class="metric-icon"><Clock3 size={24} /></span>
    <div>
      <small>Waktu belajar minggu ini</small>
      <strong>
        {stats.minutes}
        <em>/ {user.weeklyMinutes} menit</em>
      </strong>
    </div>
  </div>
  <div>
    <span class="metric-icon peach"><BookOpen size={24} /></span>
    <div>
      <small>Sesi selesai minggu ini</small>
      <strong>
        {stats.lessons}
        <em>/ {user.weeklyLessons} target</em>
      </strong>
    </div>
  </div>
  <div>
    <span class="metric-icon lavender"><Target size={24} /></span>
    <div>
      <small>Ketepatan jawaban</small>
      <strong>
        {stats.accuracy}
        <em>% minggu ini</em>
      </strong>
    </div>
  </div>
</div>
<div class="dashboard-grid">
  <section class="panel goal-panel">
    <div class="section-head">
      <h2>Ke mana kamu ingin melangkah?</h2>
      <Target size={23} />
    </div>
    <span class="eyebrow">TARGET PRIBADIMU</span>
    <p class="goal-text">{user.goal}</p>
    <p class="muted">
      Contoh, bacaan, dan latihan akan diarahkan untuk membantu mencapai target ini.
    </p>
    <button class="text-btn" on:click={() => navigate('profile')}>
      Sesuaikan target <ArrowUpRight size={17} />
    </button>
  </section>
  <section class="panel">
    <div class="section-head">
      <h2>Little wins this week.</h2>
      <Leaf size={22} />
    </div>
    <div class="week-dots">
      {#each weekDays as day}
        <div>
          <span class:done={day.done}>
            {#if day.done}
              <Check size={18} />
            {:else}
              {day.day}
            {/if}
          </span>
          <small>{day.label}</small>
        </div>
      {/each}
    </div>
    <div class="progress-track">
      <div style:width={`${Math.min(100, (stats.lessons / user.weeklyLessons) * 100)}%`}></div>
    </div>
    <div class="progress-label">
      <strong>{stats.lessons} dari {user.weeklyLessons} sesi</strong>
      <span>
        {stats.lessons >= user.weeklyLessons ? 'Target tercapai ✳' : 'Satu langkah lagi.'}
      </span>
    </div>
  </section>
</div>
<section class="panel skill-panel">
  <div class="section-head">
    <h2>Yang sedang kamu tumbuhkan</h2>
    <span class="muted">Hasil latihan minggu ini</span>
  </div>
  <div class="skill-grid">
    {#each Object.entries(skills) as [id, label]}
      {@const score = stats.skills?.[id] || {
        correct: 0,
        total: 0,
      }}
      <div>
        <div>
          <span>{label}</span>
          <small>
            {score.total ? `${score.correct}/${score.total} benar` : 'Belum ada latihan'}
          </small>
        </div>
        <div class="progress-track">
          <div style:width={`${score.total ? (score.correct / score.total) * 100 : 0}%`}></div>
        </div>
      </div>
    {/each}
  </div>
</section>
