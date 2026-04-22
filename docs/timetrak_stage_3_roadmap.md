# TimeTrak Stage 3 Roadmap

> **This is orientation, not a plan.**
> Every candidate listed here is a sketch, not a commitment. Ordering is not binding. Candidates may be removed, re-ordered, or abandoned at any time. **Every candidate MUST go through `/opsx:propose <change-name>` before any implementation begins** — no Stage 3 work starts from this doc directly. See `CLAUDE.md` and `docs/openspec_flow_nothing_to_mvp_to_real_product.md#stage-3--real-product` for the operating posture.

The purpose of this document is to give a new contributor (human or AI) a one-screen view of where TimeTrak could plausibly go next, so the next `/opsx:propose` isn't starting from a blank page. It is *not* a backlog, ticket tracker, or product spec. `openspec/specs/` remains the source of truth for what the product does today; per-change proposals under `openspec/changes/` remain the source of truth for what is actively being built.

## Likely next

Short, well-scoped candidates that build directly on the current baseline (workspace, clients, projects, tracking, rates, reporting). Each would be one or two focused OpenSpec changes.

- **CSV export for time entries and reporting.** Users already filter entries and reports by date range, client, and project; exporting those same views as CSV is a natural extension and the lowest-risk Stage 3 feature. Server-rendered streaming response, one export endpoint per existing read view, no new domain concepts. Primary spec touch: `reporting` and `tracking`.
- **Invoice generation from rate snapshots.** The rates capability already stamps an immutable `hourly_rate_minor` and `currency_code` onto every closed time entry. Grouping those snapshotted entries by client and date range into a PDF or HTML invoice is the highest-value Stage 3 capability and flows cleanly from the existing read model. New `invoices` capability likely; reads snapshot-only from `time_entries`, never calls rate resolution.
- **Team workspaces with non-owner roles.** The workspace capability was authored MVP-first for solo freelancers but structured for multi-member teams (the `workspace_members.role` column exists, bootstrap provisions `owner` on signup). Stage 3 candidates include inviting members, a non-owner `member` role with read/write-own scope, and admin transfers. Primary spec touch: `workspace` and `auth`; cross-workspace authorization contract already covers the hard part.

## Later / exploratory

Broader candidates whose scope, UX, or technical bet has not been pressure-tested yet. Would likely split into multiple changes, and several could be dropped entirely.

- **Native mobile or PWA timer.** A quick-start/stop timer surface outside the full web UI, either as a PWA over the existing HTMX app or a thin native client. Would need a public timer API if it leaves the server-rendered model.
- **Email and webhook notifications.** Forgotten running timers, weekly reports, invoice delivery. Requires outbound mail infrastructure and notification preferences. Keep in mind the Stage 1 rule against unvetted new dependencies.
- **Audit log.** An append-only record of who changed which entry, rate rule, or client. Overlaps with compliance story for team workspaces; most useful once non-owner roles exist.
- **Data import from competitor tools.** Harvest/Toggl/Clockify CSV ingestion to lower switch costs. Low engineering risk but editorial-heavy — each source has its own quirks and the mapping decisions need product input.

## How candidates graduate and how this doc is kept current

- A candidate graduates when `/opsx:propose <change-name>` is invoked for it. At that point the candidate's paragraph is removed from this doc in the same commit that lands the change — the change's own `proposal.md` is now the source of truth for that work.
- A candidate is abandoned when we explicitly decide not to build it. The paragraph is removed (optionally with a one-line note under a `## Abandoned` section if the reasoning is worth preserving for the next reviewer) in whichever subsequent change happens to touch this doc.
- This doc is allowed to shrink. A short roadmap is better than a stale one.
- This doc is **not** a behavioral spec and **must not** be treated as one. It does not define accepted behavior, it does not bind implementation, and it does not replace `/opsx:propose`.
