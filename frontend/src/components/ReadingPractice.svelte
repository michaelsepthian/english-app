<script>
  import { onMount, onDestroy } from 'svelte';
  import { Mic, Square, Volume2, LoaderCircle, CheckCircle2, RotateCcw } from '@lucide/svelte';
  import { api } from '../api';
  export let item;
  export let lessonId;
  export let pronunciationAvailable = false;
  export let onPlay = () => {};
  let index = 0,
    recording = false,
    busy = false,
    error = '',
    result = null,
    seconds = 0,
    recognizer = null,
    recorder = null,
    stream = null,
    timer,
    clock,
    audioUrl = '';
  $: sentences = (item.passage.match(/[^.!?]+(?:[.!?]+|$)/g) || [item.passage])
    .map((s) => s.trim())
    .filter(Boolean);
  $: reference = sentences[index] || sentences[0];
  $: supported =
    typeof window !== 'undefined' &&
    (pronunciationAvailable
      ? !!navigator.mediaDevices?.getUserMedia
      : !!(window.SpeechRecognition || window.webkitSpeechRecognition));
  onMount(() => {
    api(`reading/checks?lessonId=${lessonId}`)
      .then((r) => {
        result = r.checks.find((c) => c.sourceId === item.id) || null;
        if (result) index = Math.max(0, sentences.indexOf(result.reference));
      })
      .catch(() => {});
  });
  onDestroy(() => {
    cleanup();
    if (audioUrl) URL.revokeObjectURL(audioUrl);
  });
  function cleanup() {
    clearTimeout(timer);
    clearInterval(clock);
    if (recognizer) {
      recognizer.onend = null;
      recognizer.onerror = null;
      recognizer.onresult = null;
      recognizer.abort();
      recognizer = null;
    }
    if (recorder) {
      recorder.onstop = null;
      if (recorder.state === 'recording') recorder.stop();
      recorder = null;
    }
    stream?.getTracks().forEach((t) => t.stop());
    stream = null;
    recording = false;
  }
  async function checkTranscript(text) {
    busy = true;
    try {
      const r = await api('reading/check', {
        method: 'POST',
        body: { lessonId, sourceId: item.id, sentenceIndex: index, transcript: text },
      });
      result = r.check;
    } catch (e) {
      error = e.message;
    } finally {
      busy = false;
    }
  }
  async function wav(blob) {
    const Audio = window.AudioContext || window.webkitAudioContext;
    const ctx = new Audio();
    try {
      const decoded = await ctx.decodeAudioData(await blob.arrayBuffer());
      const duration = Math.min(decoded.duration, 30);
      const offline = new OfflineAudioContext(1, Math.ceil(duration * 16000), 16000);
      const source = offline.createBufferSource();
      source.buffer = decoded;
      source.connect(offline.destination);
      source.start();
      const data = (await offline.startRendering()).getChannelData(0);
      const buffer = new ArrayBuffer(44 + data.length * 2);
      const view = new DataView(buffer);
      const text = (offset, value) => {
        for (let i = 0; i < value.length; i++) view.setUint8(offset + i, value.charCodeAt(i));
      };
      text(0, 'RIFF');
      view.setUint32(4, 36 + data.length * 2, true);
      text(8, 'WAVE');
      text(12, 'fmt ');
      view.setUint32(16, 16, true);
      view.setUint16(20, 1, true);
      view.setUint16(22, 1, true);
      view.setUint32(24, 16000, true);
      view.setUint32(28, 32000, true);
      view.setUint16(32, 2, true);
      view.setUint16(34, 16, true);
      text(36, 'data');
      view.setUint32(40, data.length * 2, true);
      for (let i = 0; i < data.length; i++) {
        const n = Math.max(-1, Math.min(1, data[i]));
        view.setInt16(44 + i * 2, n < 0 ? n * 0x8000 : n * 0x7fff, true);
      }
      return new Blob([buffer], { type: 'audio/wav' });
    } finally {
      await ctx.close();
    }
  }
  async function start() {
    error = '';
    result = null;
    if (!supported) {
      error =
        'Latihan suara belum tersedia di browser ini. Gunakan Chrome atau Edge yang mendukung mikrofon dan pengenal ucapan.';
      return;
    }
    try {
      if (pronunciationAvailable) {
        stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        const mime = ['audio/webm', 'audio/mp4', 'audio/ogg'].find((t) =>
          MediaRecorder.isTypeSupported(t)
        );
        const current = new MediaRecorder(stream, mime ? { mimeType: mime } : {});
        recorder = current;
        const chunks = [];
        current.ondataavailable = (e) => {
          if (e.data.size) chunks.push(e.data);
        };
        current.onstop = async () => {
          const blob = new Blob(chunks, { type: current.mimeType });
          cleanup();
          busy = true;
          try {
            if (audioUrl) URL.revokeObjectURL(audioUrl);
            audioUrl = URL.createObjectURL(blob);
            const data = new FormData();
            data.append('lessonId', lessonId);
            data.append('sourceId', item.id);
            data.append('sentenceIndex', String(index));
            data.append('audio', await wav(blob), 'reading.wav');
            result = (await api('reading/check', { method: 'POST', body: data })).check;
          } catch (e) {
            error = e.message;
          } finally {
            busy = false;
          }
        };
        current.start();
      } else {
        const Recognition = window.SpeechRecognition || window.webkitSpeechRecognition;
        recognizer = new Recognition();
        recognizer.lang = 'en-US';
        recognizer.continuous = false;
        recognizer.interimResults = false;
        recognizer.onresult = (e) => {
          const text = Array.from(e.results)
            .map((r) => r[0].transcript)
            .join(' ');
          cleanup();
          checkTranscript(text);
        };
        recognizer.onerror = (e) => {
          if (e.error !== 'aborted')
            error =
              'Ucapan belum terbaca. Izinkan mikrofon, lalu coba di tempat yang lebih tenang.';
          cleanup();
        };
        recognizer.onend = () => cleanup();
        recognizer.start();
      }
      recording = true;
      seconds = 0;
      clock = setInterval(() => seconds++, 1000);
      timer = setTimeout(stop, 30000);
    } catch (e) {
      cleanup();
      error = 'Mikrofon belum dapat digunakan. Periksa izin browser dan coba lagi.';
    }
  }
  function stop() {
    if (recorder?.state === 'recording') recorder.stop();
    else recognizer?.stop();
  }
</script>

<section class="reading-practice">
  <div class="section-head">
    <div>
      <span class="eyebrow">YOUR TURN TO SPEAK</span>
      <h3>Yuk, baca dengan lantang.</h3>
    </div>
    <Mic size={24} />
  </div>
  <label for={`sentence-${item.id}`}>Pilih kalimat untuk dilatih</label>
  <select
    id={`sentence-${item.id}`}
    bind:value={index}
    disabled={recording || busy}
    on:change={() => (result = null)}
  >
    {#each sentences as sentence, i}
      <option value={i}>
        Kalimat {i + 1} · {sentence.slice(0, 70)}{sentence.length > 70 ? '…' : ''}
      </option>
    {/each}
  </select>
  <blockquote>{reference}</blockquote>
  <div class="reading-actions">
    <button class="secondary" on:click={() => onPlay(reference)}>
      <Volume2 size={18} /> Dengarkan contoh
    </button>

    {#if recording}
      <button class="primary recording" on:click={stop}>
        <Square size={18} /> Selesai · {seconds}s
      </button>
    {:else}
      <button class="primary" disabled={busy || !supported} on:click={start}>
        {#if busy}
          <LoaderCircle class="spin" size={18} /> Memeriksa bacaan…
        {:else}
          <Mic size={18} />
          {pronunciationAvailable ? 'Rekam & periksa pelafalan' : 'Baca & cocokkan kata'}
        {/if}
      </button>
    {/if}
  </div>
  <p class="caption">
    {pronunciationAvailable
      ? 'Rekam satu kalimat, maksimal 30 detik. Rekaman diperiksa lalu tidak disimpan.'
      : 'Mode ini memeriksa kecocokan kata yang dikenali browser, bukan ketepatan bunyi atau aksen. Skor pelafalan suara belum tersedia.'}
  </p>

  {#if !supported}
    <p class="caption">
      Latihan suara belum didukung browser ini. Kamu tetap bisa mendengarkan contoh dan berlatih
      sendiri.
    </p>
  {/if}

  {#if error}
    <p class="practice-error" role="alert">{error}</p>
  {/if}

  {#if audioUrl}
    <audio controls src={audioUrl} aria-label="Rekaman bacaanmu"></audio>
  {/if}

  {#if result}
    <div class="reading-result enter">
      <div class="reading-score">
        <strong>
          {Math.round(result.score)}
          <small>/100</small>
        </strong>
        <div>
          <b>{result.label}</b>
          <p>
            {result.mode === 'words'
              ? 'Perbandingan kata hasil ucapan'
              : 'Hasil pemeriksaan rekaman suara'}
          </p>
        </div>
      </div>

      {#if result.mode === 'pronunciation'}
        <div class="pronunciation-stats">
          <span>
            Akurasi <b>{Math.round(result.accuracy)}</b>
          </span>
          <span>
            Kelancaran <b>{Math.round(result.fluency)}</b>
          </span>
          <span>
            Kelengkapan <b>{Math.round(result.completeness)}</b>
          </span>
        </div>
      {/if}

      <p>
        <small>YANG TERBACA</small>
        <br />
        {result.transcript}
      </p>
      <div class="word-marks">
        {#each result.words as word}
          <span
            class:needs-work={word.status !== 'correct'}
            title={word.heard && word.heard !== word.word ? `Terbaca: ${word.heard}` : word.status}
          >
            {word.word}
          </span>
        {/each}
      </div>
      <p class="reading-feedback">{result.feedback}</p>
      <button class="text-btn" disabled={recording || busy || !supported} on:click={start}>
        <RotateCcw size={16} /> Coba sekali lagi
      </button>
    </div>
  {/if}
</section>
