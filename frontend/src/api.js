export async function api(path, { method = 'GET', body, signal } = {}) {
  const response = await fetch(`/api/${path}`, {
    method,
    credentials: 'same-origin',
    signal,
    headers:
      body instanceof FormData
        ? {}
        : body !== undefined
          ? { 'Content-Type': 'application/json' }
          : {},
    body: body instanceof FormData ? body : body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (!response.ok) {
    const result = await response.json().catch(() => ({}));
    throw new Error(result.error || 'Permintaan gagal. Coba lagi.');
  }
  return response.headers.get('content-type')?.includes('application/json')
    ? response.json()
    : response.blob();
}
export const tenses = [
  'Simple Present',
  'Present Continuous',
  'Simple Past',
  'Present Perfect',
  'Simple Future',
  'Past Continuous',
  'Past Perfect',
  'Present Perfect Continuous',
  'Past Perfect Continuous',
  'Future Continuous',
  'Future Perfect',
  'Future Perfect Continuous',
];
export const skills = {
  vocabulary: 'Kosakata',
  tenses: 'Tenses',
  reading: 'Reading',
  listening: 'Listening',
};
