<script>
  // MIVA-GOAL: rendering only; App owns session state and side effects.
  import { LogOut, Check, CalendarDays, LoaderCircle } from '@lucide/svelte';
  import { tenses } from '../api';
  export let profile;
  export let user;
  export let greeting;
  export let busy;
  export let saveProfile;
  export let toggleDay;
  export let calendar;
  export let logout;
</script>

<div class="heading">
  <div>
    <span class="eyebrow">YOUR PACE. YOUR DIRECTION.</span>
    <h1>
      Tujuanmu,
      <br />
      <em>arah belajarmu.</em>
    </h1>
    <p>Target ini menjadi acuan materi, contoh kalimat, dan latihan di setiap sesi.</p>
  </div>
</div>
<div class="settings-grid">
  <section class="panel">
    <h2>Rencana belajarmu</h2>
    <form on:submit|preventDefault={saveProfile}>
      <div class="two-col">
        <div>
          <label for="profile-name">Nama</label>
          <input
            id="profile-name"
            bind:value={profile.name}
            required
            minlength="2"
            maxlength="60"
          />
        </div>
        <div>
          <label for="level">Level bahasa Inggris</label>
          <select id="level" bind:value={profile.level}>
            <option value="A1">A1 · Baru mulai</option>
            <option value="A2">A2 · Dasar</option>
            <option value="B1">B1 · Menengah</option>
            <option value="B2">B2 · Cukup lancar</option>
          </select>
        </div>
      </div>
      <label for="goal">Target belajarmu</label>
      <textarea
        id="goal"
        bind:value={profile.goal}
        rows="4"
        required
        minlength="5"
        maxlength="300"
        placeholder="Contoh: Saya ingin percaya diri saat rapat dengan klien luar negeri."
      ></textarea>
      <p class="caption">
        Tulis situasi yang ingin kamu kuasai. Semakin spesifik, semakin relevan latihanmu.
      </p>
      <div class="two-col">
        <div>
          <label for="weekly-minutes">Menit per minggu</label>
          <input
            id="weekly-minutes"
            type="number"
            bind:value={profile.weeklyMinutes}
            min="15"
            max="1200"
            required
          />
        </div>
        <div>
          <label for="weekly-lessons">Sesi per minggu</label>
          <input
            id="weekly-lessons"
            type="number"
            bind:value={profile.weeklyLessons}
            min="1"
            max="7"
            required
          />
        </div>
      </div>
      <label for="default-tense">Tenses awal</label>
      <select id="default-tense" bind:value={profile.tense}>
        {#each tenses as tense}
          <option>{tense}</option>
        {/each}
      </select>
      <label for="reminder">Jam belajar (WIB)</label>
      <input id="reminder" type="time" bind:value={profile.reminder} required />
      <p class="field-label">Hari belajar</p>
      <div class="day-buttons">
        {#each ['Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab', 'Min'] as label, i}
          <button
            type="button"
            class:active={profile.days.includes(i)}
            aria-pressed={profile.days.includes(i)}
            on:click={() => toggleDay(i)}
          >
            {label}
          </button>
        {/each}
      </div>
      <button class="primary" disabled={!!busy}>
        {#if busy === 'profile'}
          <LoaderCircle class="spin" size={19} />
        {:else}
          Simpan target <Check size={19} />
        {/if}
      </button>
    </form>
  </section>
  <div class="settings-side">
    <section class="panel">
      <span class="empty-icon small"><CalendarDays size={27} /></span>
      <h2>Waktu kecil untuk dirimu.</h2>
      <p class="muted">Simpan rencanamu, lalu tambahkan jadwal belajar ke kalender favoritmu.</p>
      <button class="secondary" on:click={calendar}>
        <CalendarDays size={19} /> Unduh pengingat kalender
      </button>
      <p class="caption">
        Impor ke kalender dan aktifkan notifikasi. Saat mengubah jadwal, perbarui acara lama agar
        tidak ganda.
      </p>
    </section>
    <section class="panel account">
      <span class="avatar large">{greeting[0]?.toUpperCase()}</span>
      <h2>{user.name}</h2>
      <p>{user.email}</p>
      <p class="muted">Progres dan sesi belajarmu tersimpan di akun ini.</p>
      <button class="secondary" on:click={logout}><LogOut size={19} /> Keluar dari akun</button>
    </section>
  </div>
</div>
