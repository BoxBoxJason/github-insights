"use strict";

/* ------------------------------------------------------------------ *
 * github-insights — static Steam-themed viewer
 *
 * Reads window.REPORTS (an array of RepoActivity objects produced by
 * generate.sh) and renders a Steam-style library + per-repo detail view.
 * No framework, no build step — works over file://.
 * ------------------------------------------------------------------ */

const REPORTS = Array.isArray(window.REPORTS) ? window.REPORTS : null;

/* Activity categories: key on the RepoActivity object -> display config.
 * `colorClass` matches the CSS bar/dot colors. */
const CATEGORIES = [
  {
    key: "prs_authored",
    label: "PRs Authored",
    short: "PRs",
    color: "c-prs",
    render: renderPR,
  },
  {
    key: "prs_reviewed",
    label: "PRs Reviewed",
    short: "Reviews",
    color: "c-reviews",
    render: renderReviewedPR,
  },
  {
    key: "issues_created",
    label: "Issues Created",
    short: "Issues",
    color: "c-issues",
    render: renderIssue,
  },
  {
    key: "issues_commented",
    label: "Issues Commented",
    short: "Comments",
    color: "c-comments",
    render: renderIssue,
  },
  {
    key: "maintainer_new_issues",
    label: "New Issues (Maintainer)",
    short: "Triage",
    color: "c-issues",
    render: renderIssue,
  },
  {
    key: "mentions",
    label: "Mentions",
    short: "Mentions",
    color: "c-mentions",
    render: renderMention,
  },
  {
    key: "releases",
    label: "Releases",
    short: "Releases",
    color: "c-releases",
    render: renderRelease,
  },
  {
    key: "tags",
    label: "Tags",
    short: "Tags",
    color: "c-tags",
    render: renderTag,
  },
];

const app = document.getElementById("app");

/* -------------------------- helpers -------------------------- */

function esc(s) {
  if (s == null) return "";
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

/* Escape text, then turn bare URLs into safe links. */
function linkify(s) {
  const escaped = esc(s);
  return escaped.replace(/(https?:\/\/[^\s<]+)/g, (m) => {
    const url = m.replace(/[.,;:)]+$/, "");
    const trail = m.slice(url.length);
    return `<a href="${url}" target="_blank" rel="noopener">${url}</a>${trail}`;
  });
}

function fmtDate(s) {
  if (!s) return "";
  const d = new Date(s);
  if (isNaN(d)) return esc(s);
  return d.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function fmtDateShort(s) {
  if (!s) return "";
  const d = new Date(s);
  if (isNaN(d)) return "";
  return d.toLocaleDateString(undefined, { month: "short", year: "numeric" });
}

function count(repo, key) {
  const v = repo[key];
  return Array.isArray(v) ? v.length : 0;
}

function repoTotal(repo) {
  return CATEGORIES.reduce((n, c) => n + count(repo, c.key), 0);
}

function repoKey(repo) {
  return `${repo.owner}/${repo.name}`;
}

function statusChip(status) {
  const s = (status || "").toLowerCase();
  let cls = "neutral";
  if (s.includes("merged")) cls = "merged";
  else if (s.includes("await") || s.includes("open") || s.includes("review"))
    cls = "open";
  else if (s.includes("draft")) cls = "draft";
  else if (s.includes("closed")) cls = "closed";
  return `<span class="chip ${cls}"><span class="chip-dot"></span>${esc(status || "—")}</span>`;
}

function reviewStateChip(state) {
  const s = (state || "").toUpperCase();
  let cls = "neutral",
    label = state || "";
  if (s === "APPROVED") {
    cls = "approved";
    label = "Approved";
  } else if (s === "CHANGES_REQUESTED") {
    cls = "changes";
    label = "Changes";
  } else if (s === "COMMENTED") {
    cls = "commented";
    label = "Commented";
  } else if (s === "DISMISSED") {
    cls = "closed";
    label = "Dismissed";
  }
  return `<span class="chip ${cls}">${esc(label)}</span>`;
}

/* Collapsible body + optional comments, rendered lazily-safe (escaped). */
function expander(label, bodyHtml) {
  if (!bodyHtml) return "";
  return `<div class="expander">
    <button class="expander-toggle" data-expand>▸ ${esc(label)}</button>
    <div class="expander-body">${bodyHtml}</div>
  </div>`;
}

function commentsBlock(comments) {
  if (!Array.isArray(comments) || comments.length === 0) return "";
  const items = comments
    .map(
      (c) => `
    <div class="comment">
      <div class="comment-head"><b>${esc(c.author)}</b> · ${fmtDate(c.created_at)}${c.url ? ` · <a href="${esc(c.url)}" target="_blank" rel="noopener">link</a>` : ""}</div>
      <div class="comment-body">${linkify(c.body)}</div>
    </div>`,
    )
    .join("");
  return `<div class="comments">${items}</div>`;
}

/* -------------------------- item renderers -------------------------- */

function renderPR(pr) {
  const diff = `<span class="diff"><span class="add">+${pr.additions || 0}</span> <span class="del">−${pr.deletions || 0}</span> · ${pr.changed_files || 0} file${pr.changed_files === 1 ? "" : "s"}</span>`;
  const reviewers =
    Array.isArray(pr.reviewers) && pr.reviewers.length
      ? `<span class="reviewers">Reviewers: ${pr.reviewers.map((r) => `<b>${esc(r)}</b>`).join(", ")}</span>`
      : "";
  const merged = pr.merged_at
    ? ` · merged ${fmtDate(pr.merged_at)}${pr.merged_by ? ` by ${esc(pr.merged_by)}` : ""}`
    : "";

  let body = "";
  if (pr.description)
    body += `<div class="body-text">${linkify(pr.description)}</div>`;
  body += commentsBlock(pr.comments);

  return `<div class="card">
    <div class="card-top">
      <div class="card-title"><span class="card-num">#${pr.number}</span> ${esc(pr.title)}</div>
      ${statusChip(pr.status)}
      <a href="${esc(pr.url)}" target="_blank" rel="noopener">↗</a>
    </div>
    <div class="card-meta">
      <span class="who">by ${esc(pr.author)}</span>
      <span>opened ${fmtDate(pr.created_at)}${merged}</span>
      ${diff}
      ${reviewers}
    </div>
    ${expander(body ? "Description & comments" : "", body)}
  </div>`;
}

function renderReviewedPR(pr) {
  const reviews = Array.isArray(pr.user_reviews) ? pr.user_reviews : [];
  const states = reviews
    .map(
      (r) =>
        `${reviewStateChip(r.state)} <span style="color:var(--muted)">${fmtDate(r.submitted_at)}</span>`,
    )
    .join(" ");
  let body = "";
  reviews.forEach((r) => {
    if (r.body)
      body += `<div class="comment"><div class="comment-head">${reviewStateChip(r.state)} · ${fmtDate(r.submitted_at)}</div><div class="comment-body">${linkify(r.body)}</div></div>`;
  });

  return `<div class="card">
    <div class="card-top">
      <div class="card-title"><span class="card-num">#${pr.number}</span> ${esc(pr.title)}</div>
      ${statusChip(pr.status)}
      <a href="${esc(pr.url)}" target="_blank" rel="noopener">↗</a>
    </div>
    <div class="card-meta">
      <span class="who">authored by ${esc(pr.author)}</span>
      <span>${states || "reviewed"}</span>
    </div>
    ${body ? expander("Review comments", `<div class="comments">${body}</div>`) : ""}
  </div>`;
}

function renderIssue(issue) {
  let body = "";
  if (issue.body) body += `<div class="body-text">${linkify(issue.body)}</div>`;
  body += commentsBlock(issue.comments);
  const reactions = issue.reactions ? `<span>♥ ${issue.reactions}</span>` : "";

  return `<div class="card">
    <div class="card-top">
      <div class="card-title"><span class="card-num">#${issue.number}</span> ${esc(issue.title)}</div>
      <a href="${esc(issue.url)}" target="_blank" rel="noopener">↗</a>
    </div>
    <div class="card-meta">
      <span class="who">by ${esc(issue.author)}</span>
      <span>opened ${fmtDate(issue.created_at)}</span>
      ${reactions}
    </div>
    ${expander(body ? "Body & comments" : "", body)}
  </div>`;
}

function renderMention(m) {
  return `<div class="card">
    <div class="card-top">
      <div class="card-title">${esc(m.title)}</div>
      ${m.source ? `<span class="chip neutral">${esc(m.source)}</span>` : ""}
      ${m.url ? `<a href="${esc(m.url)}" target="_blank" rel="noopener">↗</a>` : ""}
    </div>
    <div class="card-meta">
      <span class="who">by ${esc(m.author)}</span>
      <span>${fmtDate(m.created_at)}</span>
    </div>
    ${m.body ? expander("Context", `<div class="body-text">${linkify(m.body)}</div>`) : ""}
  </div>`;
}

function renderRelease(r) {
  const chips = [];
  if (r.draft) chips.push(`<span class="chip draft">Draft</span>`);
  if (r.prerelease)
    chips.push(`<span class="chip commented">Pre-release</span>`);
  return `<div class="card">
    <div class="card-top">
      <div class="card-title">${esc(r.title || r.tag_name)} ${r.tag_name && r.tag_name !== r.title ? `<span class="card-num">${esc(r.tag_name)}</span>` : ""}</div>
      ${chips.join(" ")}
      ${r.url ? `<a href="${esc(r.url)}" target="_blank" rel="noopener">↗</a>` : ""}
    </div>
    <div class="card-meta">
      <span>published ${fmtDate(r.published_at || r.created_at)}</span>
      ${r.business_value ? `<span>${esc(r.business_value)}</span>` : ""}
    </div>
    ${r.body ? expander("Release notes", `<div class="body-text">${linkify(r.body)}</div>`) : ""}
  </div>`;
}

function renderTag(t) {
  return `<div class="card">
    <div class="card-top">
      <div class="card-title">${esc(t.name || t.title)}</div>
      ${t.commit_sha ? `<span class="chip neutral">${esc(t.commit_sha.slice(0, 7))}</span>` : ""}
      ${t.url ? `<a href="${esc(t.url)}" target="_blank" rel="noopener">↗</a>` : ""}
    </div>
    <div class="card-meta"><span>${fmtDate(t.created_at)}</span></div>
  </div>`;
}

/* -------------------------- views -------------------------- */

function renderLibrary(filter) {
  const q = (filter || "").trim().toLowerCase();
  const repos = REPORTS.filter(
    (r) =>
      !q ||
      repoKey(r).toLowerCase().includes(q) ||
      (r.name || "").toLowerCase().includes(q),
  ).sort((a, b) => repoTotal(b) - repoTotal(a));

  // Aggregate totals for the summary banner (across ALL repos, unfiltered).
  const totals = {};
  CATEGORIES.forEach((c) => {
    totals[c.key] = REPORTS.reduce((n, r) => n + count(r, c.key), 0);
  });
  const owner = mostCommonOwner();
  const win = dateWindow();

  const statTiles = [
    { num: REPORTS.length, label: "Repositories" },
    ...CATEGORIES.filter((c) => totals[c.key] > 0).map((c) => ({
      num: totals[c.key],
      label: c.label,
    })),
  ];

  const avatarHtml = owner
    ? `<img src="https://github.com/${esc(owner)}.png" alt="${esc(owner)}" class="summary-avatar-img" />`
    : `<div class="summary-avatar-fallback">?</div>`;

  const summary = `<section class="summary">
    <div class="summary-head">
      <div class="summary-avatar">${avatarHtml}</div>
      <div>
        <h1 class="summary-title">${owner ? `<strong>${esc(owner)}</strong>'s activity` : "GitHub activity"}</h1>
        <div class="summary-window">${win}</div>
      </div>
    </div>
    <div class="stat-grid">
      ${statTiles.map((t) => `<div class="stat-tile"><div class="stat-num">${t.num}</div><div class="stat-label">${esc(t.label)}</div></div>`).join("")}
    </div>
  </section>`;

  let grid;
  if (repos.length === 0) {
    grid = `<div class="notice"><h2>No repositories match “${esc(filter)}”.</h2></div>`;
  } else {
    grid = `<h2 class="section-title">Library — ${repos.length} repositor${repos.length === 1 ? "y" : "ies"}</h2>
      <div class="library">${repos.map(capsule).join("")}</div>`;
  }

  app.innerHTML = summary + grid;
  bindCapsules();
}

function capsule(repo) {
  const total = repoTotal(repo) || 1;
  const present = CATEGORIES.filter((c) => count(repo, c.key) > 0);
  const tags = present
    .map(
      (c) =>
        `<span class="tag">${esc(c.short)} <b>${count(repo, c.key)}</b></span>`,
    )
    .join("");
  const bar = present
    .map(
      (c) =>
        `<span class="${c.color}" style="width:${(count(repo, c.key) / total) * 100}%"></span>`,
    )
    .join("");

  return `<div class="capsule" data-repo="${esc(repoKey(repo))}" role="button" tabindex="0">
    <div class="capsule-banner"><span class="capsule-owner">${esc(repo.owner)}</span></div>
    <div class="capsule-body">
      <div class="capsule-name">${esc(repo.name)}</div>
      <div class="capsule-tags">${tags || '<span class="tag">no activity</span>'}</div>
      <div class="capsule-bar">${bar}</div>
      <div class="capsule-foot"><span>${repoTotal(repo)} items</span><span>view ▸</span></div>
    </div>
  </div>`;
}

function renderDetail(key) {
  const repo = REPORTS.find((r) => repoKey(r) === key);
  if (!repo) {
    renderLibrary("");
    return;
  }

  const sections = CATEGORIES.filter((c) => count(repo, c.key) > 0)
    .map((c) => {
      const items = repo[c.key].map(c.render).join("");
      return `<div class="section" data-section>
        <div class="section-bar" data-toggle-section>
          <span class="caret">▾</span>
          <h3>${esc(c.label)}</h3>
          <span class="section-count">${count(repo, c.key)}</span>
        </div>
        <div class="section-items">${items}</div>
      </div>`;
    })
    .join("");

  app.innerHTML = `
    <div class="detail-head">
      <button class="back-btn" data-back>‹ Library</button>
      <div>
        <h1 class="detail-title"><strong>${esc(repo.name)}</strong> <span class="card-num">${esc(repo.owner)}</span></h1>
        <div class="detail-sub">${repo.repo_url ? `<a href="${esc(repo.repo_url)}" target="_blank" rel="noopener">${esc(repo.repo)}</a>` : esc(repo.repo)} · ${dateWindow(repo)}</div>
      </div>
    </div>
    ${sections || '<div class="notice"><h2>No tracked activity in this repository.</h2></div>'}
  `;
  bindDetail();
}

/* -------------------------- meta helpers -------------------------- */

function mostCommonOwner() {
  // Best guess at the tracked user: the most frequent PR/issue author.
  const tally = {};
  REPORTS.forEach((r) => {
    (r.prs_authored || []).forEach((p) => {
      if (p.author) tally[p.author] = (tally[p.author] || 0) + 1;
    });
    (r.issues_created || []).forEach((i) => {
      if (i.author) tally[i.author] = (tally[i.author] || 0) + 1;
    });
  });
  let best = "",
    n = -1;
  for (const k in tally)
    if (tally[k] > n) {
      n = tally[k];
      best = k;
    }
  return best;
}

function dateWindow(repo) {
  const src = repo || REPORTS[0] || {};
  if (!src.start || !src.end) return "";
  return `${fmtDate(src.start)} – ${fmtDate(src.end)}`;
}

function renderFooter() {
  const el = document.getElementById("footer-meta");
  if (!el || !REPORTS || REPORTS.length === 0) return;
  const gen = REPORTS[0].generated_at;
  el.innerHTML = `Generated ${gen ? fmtDate(gen) : "—"} · ${REPORTS.length} repositories · github-insights`;
}

/* -------------------------- events / routing -------------------------- */

function bindCapsules() {
  document.querySelectorAll(".capsule").forEach((el) => {
    const go = () => {
      location.hash = `#/repo/${el.dataset.repo}`;
    };
    el.addEventListener("click", go);
    el.addEventListener("keydown", (e) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        go();
      }
    });
  });
}

function bindDetail() {
  const back = app.querySelector("[data-back]");
  if (back)
    back.addEventListener("click", () => {
      location.hash = "#/";
    });

  app.querySelectorAll("[data-toggle-section]").forEach((bar) => {
    bar.addEventListener("click", () =>
      bar.closest("[data-section]").classList.toggle("collapsed"),
    );
  });

  app.querySelectorAll("[data-expand]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const exp = btn.closest(".expander");
      exp.classList.toggle("open");
      const open = exp.classList.contains("open");
      btn.textContent = (open ? "▾ " : "▸ ") + btn.textContent.slice(2);
    });
  });
}

function route() {
  if (!REPORTS) {
    renderError();
    return;
  }
  const hash = location.hash || "#/";
  const m = hash.match(/^#\/repo\/(.+)$/);
  const search = document.getElementById("search");
  if (m) {
    if (search) search.style.visibility = "hidden";
    renderDetail(decodeURIComponent(m[1]));
    window.scrollTo(0, 0);
  } else {
    if (search) search.style.visibility = "visible";
    renderLibrary(search ? search.value : "");
  }
}

function renderError() {
  app.innerHTML = `<div class="notice">
    <h2>No data loaded</h2>
    <p>Generate the data bundle, then reload this page:</p>
    <p><code>make frontend-data</code> &nbsp;or&nbsp; <code>sh frontend/generate.sh</code></p>
  </div>`;
}

/* -------------------------- init -------------------------- */

window.addEventListener("hashchange", route);

const searchInput = document.getElementById("search");
if (searchInput) {
  searchInput.addEventListener("input", () => {
    if (!location.hash || location.hash === "#/")
      renderLibrary(searchInput.value);
  });
}

renderFooter();
route();
