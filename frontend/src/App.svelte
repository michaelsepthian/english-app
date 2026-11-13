<script>
  // MIVA-APP: coordinates account, session state, persistence and browser services.
  import AuthPage from './pages/AuthPage.svelte';
  import HomePage from './pages/HomePage.svelte';
  import StudyPlannerPage from './pages/StudyPlannerPage.svelte';
  import StudyPage from './pages/StudyPage.svelte';
  import WordsPage from './pages/WordsPage.svelte';
  import ChatPage from './pages/ChatPage.svelte';
  import SettingsPage from './pages/SettingsPage.svelte';
  import AudioPlayer from './components/AudioPlayer.svelte';
  import ChatVoiceSettings from './components/ChatVoiceSettings.svelte';
  import { onMount, onDestroy, tick } from 'svelte';
  import {
    Sprout,
    LayoutDashboard,
    BookOpen,
    Library,
    MessagesSquare,
    Target,
    LogOut,
    ChevronRight,
    Check,
    Headphones,
    LoaderCircle,
    AlertCircle,
    X,
    Leaf,
    CheckCircle2,
  } from '@lucide/svelte';
  import { api } from './api';
  // Account, navigation and transient page state.
  let ready = false,
    user = null,
    register = false,
    email = '',
    password = '',
    name = '',
    busy = '',
    error = '',
    notice = '',
    page = 'home',
    section = 'vocabulary';
  // Persisted learning data and editable account settings.
  let stats = { lessons: 0, minutes: 0, accuracy: 0, days: [], skills: {}, weekStart: '' },
    lesson = null,
    words = [],
    messages = [],
    answers = {},
    profile = null,
    query = '',
    verbsOnly = false;
  // Daily plan and background preparation state (MIVA-PLAN).
  let planner = {
      vocabulary: 5,
      tenses: ['Simple Present'],
      reading: 1,
      listening: 1,
      questions: 10,
    },
    requestId = '',
    job = null,
    jobTimer,
    jobPolling = false,
    historyItems = [],
    showHistory = false;
  // Draft persistence and active-study timing (MIVA-STUDY).
  let pronunciationAvailable = false,
    chatText = '',
    pendingChat = '',
    failedChats = [],
    draftChain = Promise.resolve(),
    draftError = false,
    heartbeatId = '',
    heartbeatTimer;
  let chatKeyboardOpen = false,
    chatViewportHeight = '',
    chatViewportTop = 0;
  $: failedChat = failedChats[0] || '';
  // Device speech playback and dictation (MIVA-AUDIO).
  let speechAvailable = false,
    voices = [],
    voiceName = '',
    speechRate = 0.85,
    audioText = '',
    speaking = false,
    paused = false,
    speechSerial = 0,
    audioBusy = false,
    playerOpen = false;
  let recognizer = null,
    recording = false,
    dictationStopping = false,
    recordSeconds = 0,
    recordClock,
    recordTimer,
    dictationLang = 'en-US',
    recognitionAvailable = false;
  const nav = [
    { id: 'home', label: 'Hari ini', mobile: 'Beranda', icon: LayoutDashboard },
    { id: 'study', label: 'Ruang belajar', mobile: 'Belajar', icon: BookOpen },
    { id: 'words', label: 'Kamus & kata kerja', mobile: 'Kamus', icon: Library },
    { id: 'chat', label: 'Ngobrol dengan AI', mobile: 'Ngobrol', icon: MessagesSquare },
    { id: 'profile', label: 'Target & jadwal', mobile: 'Target', icon: Target },
  ];
  // Derived UI values; the backend still validates all submitted counts.
  $: minimum =
    (Number(planner.reading) || 0) +
    (Number(planner.listening) || 0) +
    planner.tenses.length +
    (Number(planner.vocabulary) > 0 ? 1 : 0);
  $: estimatedMinutes = Math.round(
    planner.vocabulary * 0.7 +
      planner.tenses.length * 6 +
      planner.reading * 4 +
      planner.listening * 4 +
      planner.questions * 0.7
  );
  $: filteredWords = words.filter(
    (w) =>
      (!verbsOnly || w.v1) &&
      `${w.word} ${w.meaning} ${w.v1} ${w.v2} ${w.v3}`.toLowerCase().includes(query.toLowerCase())
  );
  $: activeSections = [
    ...(lesson?.vocabulary?.length ? [{ id: 'vocabulary', label: 'Kosakata', icon: Library }] : []),
    ...(lesson?.tenses?.length ? [{ id: 'tenses', label: 'Tenses', icon: BookOpen }] : []),
    ...(lesson?.readings?.length ? [{ id: 'reading', label: 'Reading', icon: BookOpen }] : []),
    ...(lesson?.listenings?.length
      ? [{ id: 'listening', label: 'Listening', icon: Headphones }]
      : []),
    { id: 'quiz', label: 'Latihan', icon: CheckCircle2 },
  ];
  $: answered = Object.keys(answers).length;
  $: greeting = user?.name?.split(' ')[0] || '';
  $: weekDays = Array.from({ length: 7 }, (_, i) => {
    const d = new Date(
      `${stats.weekStart || new Date().toISOString().slice(0, 10)}T12:00:00+07:00`
    );
    d.setDate(d.getDate() + i);
    const key = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Jakarta' }).format(d);
    return {
      label: ['Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab', 'Min'][i],
      day: d.getDate(),
      key,
      done: stats.days.includes(key),
    };
  });
  $: if (ready) syncHeartbeat(page === 'study' && lesson && !lesson.attempt ? lesson.id : '');
  function friendly(e) {
    let message = e?.message || String(e);
    if (/API|OpenAI|AI_|provider|key|model|saldo|ChatGPT|token/i.test(message))
      message = 'Materi belum dapat disiapkan saat ini. Coba kembali beberapa saat lagi.';
    return message;
  }
  function showError(e) {
    error = friendly(e);
  }
  function flash(text) {
    notice = text;
    setTimeout(() => {
      if (notice === text) notice = '';
    }, 5000);
  }
  function setLesson(value) {
    lesson = value;
    answers = { ...(value?.draft || {}) };
    const parts = [
      ...(value?.vocabulary?.length ? ['vocabulary'] : []),
      ...(value?.tenses?.length ? ['tenses'] : []),
      ...(value?.readings?.length ? ['reading'] : []),
      ...(value?.listenings?.length ? ['listening'] : []),
      'quiz',
    ];
    if (!parts.includes(section)) section = parts[0];
  }
  // Session bootstrap: reload account, progress, dictionary, chat and history.
  async function refresh() {
    const [d, w, c, h] = await Promise.all([
      api('dashboard'),
      api('words'),
      api('chat'),
      api('lesson/history'),
    ]);
    user = d.user;
    profile = structuredClone(user);
    stats = d.stats;
    pronunciationAvailable = d.pronunciationAvailable;
    setLesson(d.lesson);
    words = w.words;
    messages = c.messages;
    historyItems = h.lessons;
    try {
      const saved = localStorage.getItem(`miva-plan-${user.id}`);
      if (saved) planner = JSON.parse(saved);
      else planner = { ...planner, tenses: [user.tense] };
    } catch {
      planner = { ...planner, tenses: [user.tense] };
    }
  }
  onMount(() => {
    speechAvailable = 'speechSynthesis' in window;
    recognitionAvailable = !!(window.SpeechRecognition || window.webkitSpeechRecognition);
    const loadVoices = () =>
      (voices = window.speechSynthesis?.getVoices().filter((v) => v.lang.startsWith('en')) || []);
    loadVoices();
    window.speechSynthesis?.addEventListener('voiceschanged', loadVoices);
    (async () => {
      try {
        const me = await api('auth/me');
        user = me.user;
        if (user) {
          await refresh();
          const result = await api('lesson/job');
          if (result.job && ['queued', 'running', 'failed'].includes(result.job.status)) {
            job = result.job;
            planner = job.plan;
            page = 'plan';
            if (job.status !== 'failed') watchJob();
          }
        }
      } catch (e) {
        showError(e);
      } finally {
        ready = true;
      }
    })();
    const visibility = () => heartbeat(document.visibilityState === 'visible');
    document.addEventListener('visibilitychange', visibility);
    return () => {
      document.removeEventListener('visibilitychange', visibility);
      window.speechSynthesis?.removeEventListener('voiceschanged', loadVoices);
    };
  });
  onDestroy(() => {
    clearInterval(jobTimer);
    stopHeartbeat();
    cleanupRecording();
    stopSpeech();
  });
  // MIVA-AUTH: account actions.
  async function authenticate() {
    if (busy) return;
    busy = 'auth';
    error = '';
    try {
      await api(register ? 'auth/register' : 'auth/login', {
        method: 'POST',
        body: { email, password, ...(register ? { name } : {}) },
      });
      password = '';
      await refresh();
      page = 'home';
    } catch (e) {
      showError(e);
    } finally {
      busy = '';
    }
  }
  async function logout() {
    try {
      await draftChain.catch(() => {});
      await api('auth/logout', { method: 'POST', body: {} });
      stopHeartbeat();
      cleanupRecording();
      stopSpeech();
      clearInterval(jobTimer);
      user = null;
      lesson = null;
      job = null;
      words = [];
      messages = [];
      pendingChat = '';
      failedChats = [];
      page = 'home';
      chatText = '';
    } catch (e) {
      showError(e);
    }
  }
  // Navigation remains here so pages do not own global session state.
  async function navigate(next) {
    if (recording) cleanupRecording();
    error = '';
    page = next === 'study' && !lesson ? 'plan' : next;
    if (next === 'profile') profile = structuredClone(user);
    await tick();
    window.scrollTo({ top: 0, behavior: 'instant' });
  }
  // MIVA-PLAN: select material, prepare it on the server, then poll its job.
  async function openPlanner() {
    await draftChain.catch(() => {});
    requestId = '';
    await navigate('plan');
  }
  function togglePart(part) {
    if (part === 'vocabulary')
      planner = { ...planner, vocabulary: planner.vocabulary === 0 ? 5 : 0 };
    if (part === 'reading') planner = { ...planner, reading: planner.reading === 0 ? 1 : 0 };
    if (part === 'listening') planner = { ...planner, listening: planner.listening === 0 ? 1 : 0 };
    if (part === 'tenses')
      planner = { ...planner, tenses: planner.tenses.length ? [] : [user.tense] };
    fixQuestions();
  }
  function toggleTense(tense) {
    if (planner.tenses.includes(tense))
      planner = { ...planner, tenses: planner.tenses.filter((t) => t !== tense) };
    else if (planner.tenses.length < 3)
      planner = { ...planner, tenses: [...planner.tenses, tense] };
    else flash('Pilih maksimal 3 tenses untuk satu sesi.');
    fixQuestions();
  }
  function fixQuestions() {
    const min =
      (Number(planner.reading) || 0) +
      (Number(planner.listening) || 0) +
      planner.tenses.length +
      (Number(planner.vocabulary) > 0 ? 1 : 0);
    const questions = planner.questions;
    if (questions != null && questions !== '') {
      planner = {
        ...planner,
        questions: Math.max(min || 1, Math.min(30, Number(questions))),
      };
    }
    requestId = '';
  }
  async function prepare() {
    if (job && ['queued', 'running'].includes(job.status)) return;
    error = '';
    if (!minimum) {
      error = 'Pilih setidaknya satu materi yang ingin dipelajari.';
      return;
    }
    busy = 'prepare';
    try {
      planner = {
        ...planner,
        vocabulary: Number(planner.vocabulary),
        reading: Number(planner.reading),
        listening: Number(planner.listening),
        questions: Number(planner.questions),
      };
      localStorage.setItem(`miva-plan-${user.id}`, JSON.stringify(planner));
      requestId = requestId || crypto.randomUUID();
      job = (await api('lesson/prepare', { method: 'POST', body: { plan: planner, requestId } }))
        .job;
      watchJob();
    } catch (e) {
      showError(e);
    } finally {
      busy = '';
    }
  }
  async function resumePreparation() {
    if (busy || !job?.canResume) return;
    busy = 'resume';
    error = '';
    try {
      job = (await api('lesson/job/resume', { method: 'POST', body: { id: job.id } })).job;
      planner = structuredClone(job.plan);
      watchJob();
    } catch (e) {
      showError(e);
    } finally {
      busy = '';
    }
  }
  function newPreparation() {
    job = null;
    requestId = '';
    error = '';
  }
  function watchJob() {
    clearInterval(jobTimer);
    pollJob();
    jobTimer = setInterval(pollJob, 2500);
  }
  async function pollJob() {
    if (jobPolling || !job?.id) return;
    jobPolling = true;
    try {
      const data = await api(`lesson/job?id=${job.id}`);
      if (data.job) job = data.job;
      if (job?.status === 'done') {
        clearInterval(jobTimer);
        setLesson(data.lesson);
        section = activeSections[0]?.id || 'vocabulary';
        words = (await api('words')).words;
        historyItems = (await api('lesson/history')).lessons;
        page = 'study';
        await tick();
        section = activeSections[0]?.id || 'quiz';
        window.scrollTo({ top: 0, behavior: 'instant' });
        flash('Sesi belajarmu siap. Selamat menikmati prosesnya!');
      }
      if (job?.status === 'failed') {
        clearInterval(jobTimer);
        error = job.error;
        requestId = '';
      }
    } catch (e) {
      showError(e);
    } finally {
      jobPolling = false;
    }
  }
  async function loadSession(id) {
    await draftChain.catch(() => {});
    error = '';
    try {
      setLesson((await api(`lesson?id=${id}`)).lesson);
      section = 'quiz';
      showHistory = false;
      await navigate('study');
    } catch (e) {
      showError(e);
    }
  }
  // MIVA-STUDY: count active time and serialize draft writes before submission.
  function syncHeartbeat(id) {
    if (id === heartbeatId) return;
    stopHeartbeat();
    if (!id) return;
    heartbeatId = id;
    heartbeat(true);
    heartbeatTimer = setInterval(() => heartbeat(document.visibilityState === 'visible'), 15000);
  }
  function heartbeat(active) {
    if (heartbeatId)
      api('lesson/heartbeat', { method: 'POST', body: { id: heartbeatId, active } }).catch(
        () => {}
      );
  }
  function stopHeartbeat() {
    clearInterval(heartbeatTimer);
    if (heartbeatId) heartbeat(false);
    heartbeatId = '';
  }
  function choose(id, index) {
    if (lesson?.attempt) return;
    answers = { ...answers, [id]: index };
    const snapshot = { ...answers },
      session = lesson.id;
    draftChain = draftChain
      .catch(() => {})
      .then(async () => {
        try {
          await api('lesson/draft', { method: 'PATCH', body: { id: session, answers: snapshot } });
          draftError = false;
        } catch (e) {
          draftError = true;
          throw e;
        }
      });
    draftChain.catch(() => {});
  }
  async function saveAnswers() {
    try {
      await draftChain.catch(() => {});
      await api('lesson/draft', { method: 'PATCH', body: { id: lesson.id, answers } });
      draftError = false;
      flash('Jawaban tersimpan. Kamu bisa melanjutkan nanti.');
    } catch (e) {
      showError(e);
    }
  }
  async function submit() {
    if (busy) return;
    busy = 'submit';
    error = '';
    try {
      await draftChain.catch(() => {});
      const r = await api('lesson/submit', { method: 'POST', body: { id: lesson.id, answers } });
      stopHeartbeat();
      setLesson(r.lesson);
      stats = r.stats;
      section = 'quiz';
      await tick();
      document
        .getElementById('session-result')
        ?.scrollIntoView({ behavior: 'smooth', block: 'start' });
      flash('Selesai! Hasil belajarmu masuk ke progres mingguan.');
    } catch (e) {
      showError(e);
    } finally {
      busy = '';
    }
  }
  async function review() {
    busy = 'review';
    try {
      setLesson((await api('lesson/review', { method: 'POST', body: { id: lesson.id } })).lesson);
      if (!lesson.attempt.review)
        flash('Rangkuman belum tersedia. Kamu tetap bisa melihat pembahasan setiap soal.');
    } catch (e) {
      showError(e);
    } finally {
      busy = '';
    }
  }
  function resultFor(id) {
    return lesson?.attempt?.results?.find((r) => r.id === id) || null;
  }
  function scorePercent() {
    return lesson?.attempt ? Math.round((lesson.attempt.score / lesson.attempt.total) * 100) : 0;
  }
  function questionsFor(source) {
    return lesson.questions.filter(
      (q) =>
        q.sourceId === source ||
        (!q.sourceId && q.skill === 'listening' && source === 'listening-1')
    );
  }
  // MIVA-AUDIO: resolve trusted material IDs before requesting speech text.
  function wordID(word) {
    return words.find((w) => w.word.toLowerCase() === word.word.toLowerCase())?.id || word.id;
  }
  async function playWord(word, part, index = 0) {
    await playAudio('word', wordID(word), '', part, index);
  }
  async function playAudio(kind, id, sourceId = '', part = '', exampleIndex = 0) {
    if (audioBusy) return;
    audioBusy = true;
    try {
      const data = await api('audio', {
        method: 'POST',
        body: { kind, id, sourceId, part, exampleIndex },
      });
      if (data instanceof Blob)
        throw new Error('Pemutar suara perangkat belum tersedia untuk materi ini.');
      speak(data.text);
    } catch (e) {
      showError(e);
    } finally {
      audioBusy = false;
    }
  }
  function speak(text) {
    if (!speechAvailable) {
      flash('Pembaca suara belum tersedia di browser ini.');
      return;
    }
    const serial = ++speechSerial;
    window.speechSynthesis.cancel();
    audioText = text;
    playerOpen = true;
    paused = false;
    const utterance = new SpeechSynthesisUtterance(text);
    utterance.lang = 'en-US';
    utterance.rate = Number(speechRate);
    const voice =
      voices.find((v) => v.voiceURI === voiceName) ||
      voices.find((v) => v.lang === 'en-US') ||
      voices[0];
    if (voice) utterance.voice = voice;
    utterance.onend = () => {
      if (serial === speechSerial) {
        speaking = false;
        paused = false;
      }
    };
    utterance.onerror = (e) => {
      if (serial === speechSerial) {
        speaking = false;
        paused = false;
        if (!['canceled', 'interrupted'].includes(e.error))
          flash('Suara belum dapat diputar. Coba pilih suara Inggris lain.');
      }
    };
    speaking = true;
    window.speechSynthesis.speak(utterance);
  }
  function pauseSpeech() {
    if (paused) {
      window.speechSynthesis.resume();
      paused = false;
    } else {
      window.speechSynthesis.pause();
      paused = true;
    }
  }
  function stopSpeech() {
    speechSerial++;
    window.speechSynthesis?.cancel();
    speaking = false;
    paused = false;
  }
  // MIVA-WORDS: add dictionary entries to the current account.
  async function lookup() {
    if (!query.trim() || busy) return;
    busy = 'lookup';
    error = '';
    try {
      const r = await api('words/lookup', { method: 'POST', body: { query: query.trim() } });
      words = r.words;
      query = r.word.word;
      flash('Kata dan contoh sudah masuk ke kamusmu.');
    } catch (e) {
      showError(e);
    } finally {
      busy = '';
    }
  }
  // MIVA-CHAT: bilingual messages and optional browser dictation.
  async function send(text = chatText) {
    text = text.trim();
    if (!text || busy || recording) return;
    if (text.length > 2000) {
      showError(new Error('Pesan maksimal 2.000 karakter. Persingkat sebelum mengirim.'));
      return;
    }
    const accountId = user?.id;
    busy = 'chat';
    error = '';
    pendingChat = text;
    chatText = '';
    try {
      const result = await api('chat', { method: 'POST', body: { text } });
      if (user?.id === accountId) messages = result.messages;
    } catch (e) {
      if (user?.id === accountId) {
        if (!chatText.trim()) chatText = text;
        else failedChats = [...failedChats, text];
        showError(e);
      }
    } finally {
      pendingChat = '';
      busy = '';
    }
  }
  function restoreFailedChat() {
    if (busy || recording || !failedChat) return;
    if (chatText.trim()) {
      flash('Kosongkan atau kirim draft saat ini sebelum memulihkan pesan.');
      return;
    }
    chatText = failedChat;
    failedChats = failedChats.slice(1);
  }
  function stopDictation() {
    if (!recognizer || dictationStopping) return;
    dictationStopping = true;
    recognizer.stop();
  }
  function startDictation() {
    if (busy || recording || page !== 'chat') return;
    if (!recognitionAvailable) {
      flash('Dikte belum tersedia. Kamu tetap bisa mengetik pesan.');
      return;
    }
    stopSpeech();
    const draft = chatText.trim();
    const results = new Map();
    let finished = false;
    try {
      const Recognition = window.SpeechRecognition || window.webkitSpeechRecognition;
      const session = new Recognition();
      recognizer = session;
      session.lang = dictationLang === 'id-ID' ? 'id-ID' : 'en-US';
      session.continuous = false;
      session.interimResults = false;
      session.onresult = (event) => {
        if (finished || recognizer !== session) return;
        for (let i = event.resultIndex; i < event.results.length; i++) {
          if (event.results[i].isFinal) results.set(i, event.results[i][0].transcript.trim());
        }
        const transcript = [...results.values()].join(' ').trim();
        chatText = [draft, transcript].filter(Boolean).join(' ');
      };
      session.onspeechend = () => {
        if (!finished && recognizer === session) stopDictation();
      };
      session.onend = () => {
        if (finished || recognizer !== session) return;
        finished = true;
        const transcript = [...results.values()].join(' ').trim();
        cleanupRecording();
        if (transcript && page === 'chat') send(chatText);
        else flash('Ucapan belum terbaca. Coba bicara lagi.');
      };
      session.onerror = (event) => {
        if (finished || recognizer !== session) return;
        finished = true;
        cleanupRecording();
        flash(
          event.error === 'not-allowed' || event.error === 'service-not-allowed'
            ? 'Izinkan mikrofon atau lanjutkan dengan mengetik.'
            : 'Ucapan belum terbaca. Coba lagi di tempat tenang.'
        );
      };
      recording = true;
      dictationStopping = false;
      recordSeconds = 0;
      session.start();
      recordClock = setInterval(() => recordSeconds++, 1000);
      recordTimer = setTimeout(() => {
        if (recognizer === session) stopDictation();
      }, 60000);
    } catch {
      cleanupRecording();
      flash('Izinkan mikrofon atau lanjutkan dengan mengetik.');
    }
  }
  function cleanupRecording() {
    clearTimeout(recordTimer);
    clearInterval(recordClock);
    if (recognizer) {
      const session = recognizer;
      recognizer = null;
      session.onend = null;
      session.onerror = null;
      session.onresult = null;
      session.onspeechend = null;
      session.abort();
    }
    recording = false;
    dictationStopping = false;
  }
  // MIVA-GOAL: account targets and calendar export.
  function toggleDay(i) {
    if (profile.days.length === 1 && profile.days.includes(i)) {
      flash('Pilih minimal satu hari belajar.');
      return;
    }
    profile = {
      ...profile,
      days: profile.days.includes(i)
        ? profile.days.filter((d) => d !== i)
        : [...profile.days, i].sort(),
    };
  }
  async function saveProfile() {
    busy = 'profile';
    error = '';
    try {
      user = (
        await api('settings', {
          method: 'PATCH',
          body: {
            name: profile.name,
            level: profile.level,
            goal: profile.goal,
            weeklyMinutes: Number(profile.weeklyMinutes),
            weeklyLessons: Number(profile.weeklyLessons),
            reminder: profile.reminder,
            days: profile.days,
            tense: profile.tense,
          },
        })
      ).user;
      profile = structuredClone(user);
      flash('Target tersimpan. Sesi berikutnya akan mengikuti arah belajarmu.');
    } catch (e) {
      showError(e);
    } finally {
      busy = '';
    }
  }
  async function calendar() {
    try {
      const blob = await api('calendar');
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'miva-jadwal.ics';
      a.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      flash('Impor jadwal ke kalender dan aktifkan pengingatnya.');
    } catch (e) {
      showError(e);
    }
  }
</script>

{#if !ready}
  <div class="loading">
    <span class="brand-mark"><Sprout size={30} /></span>
    <p>Menyiapkan ruang belajarmu…</p>
  </div>
{:else if !user}
  <AuthPage bind:register bind:email bind:password bind:name bind:error {busy} {authenticate} />
{:else}
  <div
    class="app-shell"
    class:chat-active={page === 'chat'}
    class:chat-keyboard-open={page === 'chat' && chatKeyboardOpen}
    style:--chat-viewport-height={chatViewportHeight || '100dvh'}
    style:--chat-viewport-top={`${chatViewportTop}px`}
  >
    <aside class="sidebar">
      <a class="brand" href="#home" on:click|preventDefault={() => navigate('home')}>
        <span class="brand-mark"><Sprout size={28} /></span>
        <div>
          miva
          <span class="brand-dot">.</span>
          <small>MAKE A LITTLE PROGRESS.</small>
        </div>
      </a>
      <p class="nav-label">RUANG BELAJARMU</p>
      <nav>
        {#each nav as item}
          <button
            class:active={page === item.id || (item.id === 'study' && page === 'plan')}
            title={item.label}
            on:click={() => navigate(item.id)}
          >
            <svelte:component this={item.icon} size={21} />
            <span class="nav-desktop">{item.label}</span>
            <span class="nav-mobile">{item.mobile}</span>

            {#if item.id === 'study' && lesson?.attempt}
              <Check size={16} class="nav-check" />
            {/if}
          </button>
        {/each}
      </nav>
      <div class="sidebar-bottom">
        <div class="side-quote">
          <Leaf size={25} />
          <p>
            Every little step
            <br />
            <em>is still a step.</em>
          </p>
          <small>Pelan-pelan juga sampai.</small>
        </div>
        <div class="profile">
          <span class="avatar">{greeting[0]?.toUpperCase()}</span>
          <div>
            <strong>{user.name}</strong>
            <small>{user.level} · English learner</small>
          </div>
          <button class="icon" aria-label="Keluar" on:click={logout}><LogOut size={20} /></button>
        </div>
      </div>
    </aside>
    <main class="workspace">
      <header class="topbar">
        <div>
          <span class="crumb">Ruang belajar</span>
          <ChevronRight size={15} />
          <strong>
            {page === 'plan' ? 'Rencana hari ini' : nav.find((n) => n.id === page)?.label}
          </strong>
        </div>
        <div class="topbar-right">
          {#if job && ['queued', 'running'].includes(job.status) && page !== 'plan'}
            <button class="text-btn" on:click={() => navigate('plan')}>
              <LoaderCircle size={16} class="spin" /> Sesi sedang disiapkan
            </button>
          {/if}

          <span class="level-pill">
            {user.level}
            <span>
              · {user.level === 'A1' ? 'Getting started' : 'Keep growing'}
            </span>
          </span>
        </div>
      </header>
      <div class="content">
        {#if error && page !== 'chat'}
          <div class="alert enter" role="alert">
            <AlertCircle size={21} />
            <span>{error}</span>
            <button class="icon" aria-label="Tutup pesan" on:click={() => (error = '')}>
              <X size={18} />
            </button>
          </div>
        {/if}

        {#if page === 'home'}
          <HomePage {user} {stats} {lesson} {weekDays} {greeting} {navigate} {openPlanner} />
        {:else if page === 'plan'}
          <StudyPlannerPage
            {user}
            {lesson}
            bind:planner
            bind:requestId
            {job}
            {minimum}
            {estimatedMinutes}
            {busy}
            {navigate}
            {togglePart}
            {toggleTense}
            {fixQuestions}
            {prepare}
            {resumePreparation}
            {newPreparation}
          />
        {:else if page === 'study' && lesson}
          <StudyPage
            {lesson}
            {user}
            bind:section
            bind:showHistory
            {historyItems}
            {activeSections}
            {answers}
            {answered}
            {audioBusy}
            {speechAvailable}
            {pronunciationAvailable}
            {busy}
            {draftError}
            bind:chatText
            {openPlanner}
            {loadSession}
            {resultFor}
            {scorePercent}
            {questionsFor}
            {playAudio}
            {playWord}
            {speak}
            {choose}
            {saveAnswers}
            {submit}
            {review}
            {navigate}
          />
        {:else if page === 'words'}
          <WordsPage
            {words}
            {filteredWords}
            bind:query
            bind:verbsOnly
            {busy}
            {lookup}
            {audioBusy}
            {speechAvailable}
            {playWord}
          />
        {:else if page === 'chat'}
          <ChatPage
            {messages}
            {pendingChat}
            {failedChat}
            {restoreFailedChat}
            {busy}
            bind:chatText
            bind:keyboardOpen={chatKeyboardOpen}
            bind:viewportHeight={chatViewportHeight}
            bind:viewportTop={chatViewportTop}
            {error}
            dismissError={() => (error = '')}
            {recording}
            {recordSeconds}
            {recognitionAvailable}
            {stopDictation}
            {send}
            {speak}
            {startDictation}
          >
            <svelte:fragment slot="settings">
              <ChatVoiceSettings
                bind:dictationLang
                {recording}
                {speechAvailable}
                {audioText}
                {speaking}
                {paused}
                bind:voiceName
                bind:speechRate
                {voices}
                {speak}
                {stopSpeech}
                {pauseSpeech}
              />
            </svelte:fragment>
          </ChatPage>
        {:else if page === 'profile' && profile}
          <SettingsPage
            bind:profile
            {user}
            {greeting}
            {busy}
            {saveProfile}
            {toggleDay}
            {calendar}
            {logout}
          />
        {/if}

        <footer>
          <span><Leaf size={16} /> A little more confident, every day.</span>
          <span>Made for your little wins. · miva</span>
        </footer>
      </div>
    </main>

    {#if playerOpen && audioText && page !== 'chat'}
      <AudioPlayer
        {audioText}
        bind:playerOpen
        {speaking}
        {paused}
        bind:voiceName
        bind:speechRate
        {voices}
        {speak}
        {stopSpeech}
        {pauseSpeech}
      />
    {/if}

    {#if notice}
      <div class="toast" role="status">
        <CheckCircle2 size={21} />{notice}
      </div>
    {/if}
  </div>
{/if}
