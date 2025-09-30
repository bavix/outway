export function extractDomain(raw: string): string {
  let v = (raw || '').trim();
  if (!v) return v;
  if (v.startsWith('@')) v = v.slice(1);
  try { const u = new URL(v); return u.hostname || v; } catch {}
  try { const u2 = new URL(/^https?:\/\//i.test(v) ? v : `http://${v}`); return u2.hostname || v; } catch {}
  v = v.replace(/^\w+:\/\//, '');
  v = v.replace(/\/.*$/, '');
  return v;
}

