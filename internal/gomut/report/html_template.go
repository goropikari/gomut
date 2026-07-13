package report

const htmlTemplateSource = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>gomut HTML report</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f6f4ef;
      --panel: #ffffff;
      --text: #1f2933;
      --muted: #52606d;
      --line: #d9e2ec;
      --accent: #0f766e;
      --killed: #15803d;
      --lived: #b91c1c;
      --not-covered: #64748b;
      --timed-out: #b45309;
      --not-viable: #c2410c;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      color: var(--text);
      background:
        radial-gradient(circle at top left, rgba(15, 118, 110, 0.08), transparent 32%),
        radial-gradient(circle at top right, rgba(185, 28, 28, 0.08), transparent 28%),
        var(--bg);
    }
    .page {
      max-width: 1200px;
      margin: 0 auto;
      padding: 32px 20px 48px;
    }
    .hero {
      background: linear-gradient(135deg, rgba(15, 118, 110, 0.1), rgba(255, 255, 255, 0.95));
      border: 1px solid var(--line);
      border-radius: 24px;
      padding: 28px;
      box-shadow: 0 18px 48px rgba(15, 23, 42, 0.08);
    }
    h1 {
      margin: 0 0 8px;
      font-size: clamp(2rem, 4vw, 3rem);
      letter-spacing: -0.03em;
    }
    .meta {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 12px;
      margin-top: 20px;
      color: var(--muted);
    }
    .meta div, .card, .mutation {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 18px;
    }
    .meta div {
      padding: 14px 16px;
    }
    .label {
      display: block;
      font-size: 0.78rem;
      text-transform: uppercase;
      letter-spacing: 0.12em;
      color: var(--muted);
      margin-bottom: 6px;
    }
    .summary-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
      gap: 14px;
      margin: 20px 0;
    }
    .card {
      padding: 16px;
      box-shadow: 0 10px 22px rgba(15, 23, 42, 0.04);
    }
    .card strong {
      display: block;
      font-size: 1.6rem;
      margin-top: 8px;
    }
    .score {
      border-left: 4px solid var(--accent);
    }
    .filters {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 12px;
      margin: 24px 0 18px;
      align-items: end;
    }
    .filters label {
      display: grid;
      gap: 8px;
      color: var(--muted);
      font-size: 0.92rem;
    }
    .filters input, .filters select {
      width: 100%;
      padding: 12px 14px;
      border-radius: 12px;
      border: 1px solid var(--line);
      background: #fff;
      color: var(--text);
    }
    .records {
      display: grid;
      gap: 14px;
    }
    .mutation {
      padding: 18px;
      box-shadow: 0 10px 20px rgba(15, 23, 42, 0.04);
    }
    .mutation-header {
      display: flex;
      flex-wrap: wrap;
      gap: 12px;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 14px;
    }
    .mutation-header a {
      color: var(--accent);
      text-decoration: none;
      font-weight: 600;
    }
    .pill {
      display: inline-flex;
      align-items: center;
      padding: 6px 10px;
      border-radius: 999px;
      font-size: 0.8rem;
      font-weight: 700;
      letter-spacing: 0.04em;
      text-transform: uppercase;
      color: #fff;
    }
    .result-killed { background: var(--killed); }
    .result-lived { background: var(--lived); }
    .result-not-covered { background: var(--not-covered); }
    .result-timed-out { background: var(--timed-out); }
    .result-not-viable { background: var(--not-viable); }
    .mutation-body {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
      gap: 12px;
      margin-bottom: 12px;
    }
    .code-block {
      margin: 8px 0 0;
      padding: 14px;
      border-radius: 14px;
      background: #102a43;
      color: #eff6ff;
      overflow-x: auto;
      white-space: pre;
      word-break: normal;
    }
    .message {
      margin: 0;
      color: var(--muted);
      white-space: pre-wrap;
    }
    .empty {
      padding: 32px;
      text-align: center;
      color: var(--muted);
      background: var(--panel);
      border: 1px dashed var(--line);
      border-radius: 18px;
    }
    @media print {
      body { background: #fff; }
      .hero, .card, .mutation, .meta div { box-shadow: none; }
      .filters { display: none; }
    }
  </style>
</head>
<body>
  <main class="page">
    <section class="hero">
      <h1>gomut HTML report</h1>
      <div class="meta">
        <div><span class="label">Started at</span>{{ .StartedAt }}</div>
        <div><span class="label">Target mode</span>{{ .Target.Mode }}</div>
        <div><span class="label">Target value</span>{{ .Target.Value }}</div>
        <div><span class="label">Command</span>{{ .Command }}</div>
      </div>
    </section>

    <section class="summary-grid" aria-label="Summary">
      <div class="card"><span class="label">Total</span><strong>{{ .Summary.Total }}</strong></div>
      <div class="card"><span class="label">Killed</span><strong>{{ .Summary.Killed }}</strong></div>
      <div class="card"><span class="label">Lived</span><strong>{{ .Summary.Lived }}</strong></div>
      <div class="card"><span class="label">Not covered</span><strong>{{ .Summary.NotCovered }}</strong></div>
      <div class="card"><span class="label">Timed out</span><strong>{{ .Summary.TimedOut }}</strong></div>
      <div class="card"><span class="label">Not viable</span><strong>{{ .Summary.NotViable }}</strong></div>
      <div class="card score"><span class="label">Mutation score</span><strong>{{ .MutationScore }}</strong></div>
    </section>

    <section class="filters" aria-label="Filters">
        <label>
        Result
        <select id="result-filter">
          <option value="">All results</option>
          <option value="killed">Killed</option>
          <option value="lived">Lived</option>
          <option value="not-covered">Not covered</option>
          <option value="timed-out">Timed out</option>
          <option value="not-viable">Not viable</option>
        </select>
      </label>
      <label>
        File
        <input id="file-filter" type="search" placeholder="Filter by file">
      </label>
      <label>
        Kind
        <input id="kind-filter" type="search" placeholder="Filter by mutation kind">
      </label>
    </section>

    <section class="records" id="records">
      {{ if .Records }}
        {{ range .Records }}
          <article class="mutation mutation-row" data-result="{{ .ResultLower }}" data-file="{{ .File }}" data-kind="{{ .KindLower }}">
            <header class="mutation-header">
              <div>
                <a href="{{ .Link }}" target="_blank" rel="noreferrer">{{ .File }}:{{ .Line }}</a>
                <div class="label">Kind</div>
                <div>{{ .Kind }}</div>
              </div>
              <span class="pill {{ .ResultClass }}">{{ .Result }}</span>
            </header>
            <div class="mutation-body">
              <div>
                <span class="label">Source excerpt</span>
                <pre class="code-block">{{ .Excerpt }}</pre>
              </div>
              <div>
                <span class="label">Unified diff</span>
                <pre class="code-block">{{ .Diff }}</pre>
              </div>
            </div>
            <p class="message">{{ .Message }}</p>
          </article>
        {{ end }}
      {{ else }}
        <div class="empty">No mutation records matched the selected filters.</div>
      {{ end }}
    </section>
  </main>

  <script>
    (() => {
      const resultFilter = document.getElementById("result-filter");
      const fileFilter = document.getElementById("file-filter");
      const kindFilter = document.getElementById("kind-filter");
      const rows = Array.from(document.querySelectorAll(".mutation-row"));

      const applyFilters = () => {
        const resultValue = resultFilter.value.trim().toLowerCase();
        const fileValue = fileFilter.value.trim().toLowerCase();
        const kindValue = kindFilter.value.trim().toLowerCase();

        for (const row of rows) {
          const matchesResult = !resultValue || row.dataset.result === resultValue;
          const matchesFile = !fileValue || row.dataset.file.toLowerCase().includes(fileValue);
          const matchesKind = !kindValue || row.dataset.kind.toLowerCase().includes(kindValue);
          row.hidden = !(matchesResult && matchesFile && matchesKind);
        }
      };

      resultFilter.addEventListener("change", applyFilters);
      fileFilter.addEventListener("input", applyFilters);
      kindFilter.addEventListener("input", applyFilters);
    })();
  </script>
</body>
</html>`
