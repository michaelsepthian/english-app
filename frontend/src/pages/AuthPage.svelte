<script>
  // MIVA-AUTH: rendering only; App owns session state and side effects.
  import {
    Sprout,
    Library,
    Target,
    ArrowRight,
    Headphones,
    LoaderCircle,
    AlertCircle,
    CheckCircle2,
  } from '@lucide/svelte';
  export let register;
  export let email;
  export let password;
  export let name;
  export let error;
  export let busy;
  export let authenticate;
</script>

<main class="auth-layout">
  <section class="auth-story">
    <a class="brand" href="/">
      <span class="brand-mark"><Sprout size={27} /></span>
      miva
      <span class="brand-dot">.</span>
    </a>
    <div class="auth-intro">
      <span class="eyebrow">A LITTLE PRACTICE, EVERY DAY</span>
      <h1>
        Your pace.
        <br />
        Your voice.
        <br />
        <em>Your possibilities.</em>
      </h1>
      <p>Bahasa baru membuka cerita baru. Mulai dari langkah kecil yang terasa tepat untukmu.</p>
      <div class="auth-features">
        <span><Library size={20} /> Kata yang benar-benar kamu perlukan</span>
        <span><Headphones size={20} /> Dengarkan. Pahami. Coba lagi.</span>
        <span><Target size={20} /> Belajar sesuai tujuanmu sendiri</span>
      </div>
    </div>
    <small>SMALL STEPS. A LITTLE MORE CONFIDENT.</small>
    <div class="decor-circle" aria-hidden="true"></div>
  </section>
  <section class="auth-form">
    <div class="auth-card">
      <span class="mini-leaf"><Sprout size={30} /></span>
      <span class="eyebrow">WELCOME TO YOUR LEARNING SPACE</span>
      <h2>{register ? 'Mulai bersama Miva.' : 'Senang kamu kembali.'}</h2>
      <p>
        {register
          ? 'Buat akun dan temukan ritme belajar yang cocok untukmu.'
          : 'Masuk untuk melanjutkan langkah kecilmu hari ini.'}
      </p>

      {#if error}
        <div class="alert" role="alert"><AlertCircle size={20} />{error}</div>
      {/if}

      <form on:submit|preventDefault={authenticate}>
        {#if register}
          <label for="name">Nama kamu</label>
          <input
            id="name"
            bind:value={name}
            required
            minlength="2"
            maxlength="60"
            autocomplete="name"
            placeholder="Panggilan favoritmu"
          />
        {/if}

        <label for="email">Email</label>
        <input
          id="email"
          type="email"
          bind:value={email}
          required
          autocomplete="email"
          placeholder="nama@email.com"
        />
        <label for="password">Kata sandi</label>
        <input
          id="password"
          type="password"
          bind:value={password}
          required
          minlength="10"
          maxlength="72"
          autocomplete={register ? 'new-password' : 'current-password'}
          placeholder="Minimal 10 karakter"
        />
        <button class="primary full" disabled={!!busy}>
          {#if busy === 'auth'}
            <LoaderCircle class="spin" size={19} />
          {:else}
            {register ? 'Buat akun' : 'Masuk ke Miva'}<ArrowRight size={19} />
          {/if}
        </button>
      </form>
      <p class="auth-switch">
        {register ? 'Sudah punya akun?' : 'Pertama kali di sini?'}
        <button
          on:click={() => {
            register = !register;
            error = '';
          }}
        >
          {register ? 'Masuk' : 'Buat akun'}
        </button>
      </p>
      <div class="auth-note"><CheckCircle2 size={18} /> Target dan progres tersimpan untukmu.</div>
    </div>
  </section>
</main>
