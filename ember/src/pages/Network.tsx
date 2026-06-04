import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import type { NetworkConnection, User } from "@pulse/drift/types";
import { getNetwork } from "../api/advice";
import { searchUsers } from "../api/social";
import Spinner from "../components/ui/Spinner";
import { usePageTitle } from "../hooks/usePageTitle";
import { useUiStore } from "../store/uiStore";

type NetworkDirection = "you_asked" | "you_answered" | "connected" | "nearby";

function PersonRow({
  user,
  meta,
  metaTone,
  subtitle,
  contextTags = [],
  affinity = 0,
  activeRoom,
  sharedPath,
}: {
  user: Pick<User, "id" | "handle" | "display_name">;
  meta?: string;
  metaTone?: string;
  subtitle?: string;
  contextTags?: string[];
  affinity?: number;
  activeRoom?: NetworkConnection["active_room"];
  sharedPath?: NetworkConnection["shared_path"];
}) {
  return (
    <article className="rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4 transition-colors hover:bg-[var(--color-surface)]">
      <Link to={`/profile/${user.id}`} className="flex items-start gap-3">
      <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-[var(--color-surface-active)] text-sm font-semibold text-[var(--color-text-secondary)]">
        {user.display_name?.[0] || user.handle?.[0] || "?"}
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="truncate text-sm font-semibold">
            {user.display_name || user.handle}
          </p>
          {meta && (
            <span className={`text-[11px] font-medium ${metaTone ?? ""}`}>
              {meta}
            </span>
          )}
        </div>
        <p className="mt-0.5 truncate text-xs text-[var(--color-text-muted)]">
          {subtitle ?? `@${user.handle}`}
        </p>
      </div>
      </Link>
      {(contextTags.length > 0 || activeRoom || sharedPath || affinity > 0) && (
        <div className="mt-3 space-y-2 pl-14">
          {contextTags.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {contextTags.slice(0, 4).map((tag) => (
                <span
                  key={tag}
                  className="rounded-full bg-[var(--color-surface)] px-2 py-1 text-[11px] text-[var(--color-text-secondary)]"
                >
                  #{tag.replace(/^#/, "")}
                </span>
              ))}
            </div>
          )}
          <div className="flex flex-wrap gap-2 text-[11px] text-[var(--color-text-muted)]">
            {activeRoom && (
              <span
                className="rounded-full bg-[var(--color-accent-subtle)] px-2 py-1 font-medium text-[var(--color-accent)]"
              >
                shared context · {activeRoom.member_count} nearby
              </span>
            )}
            {sharedPath && (
              <Link
                to={`/paths/${sharedPath.id}`}
                className="rounded-full bg-[var(--color-surface)] px-2 py-1 font-medium text-[var(--color-text-secondary)]"
              >
                trail · {sharedPath.title}
              </Link>
            )}
            {affinity > 0 && (
              <span className="rounded-full bg-[var(--color-surface)] px-2 py-1">
                {Math.round(affinity * 100)}% affinity
              </span>
            )}
          </div>
        </div>
      )}
    </article>
  );
}

function PeopleSearch() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<User[] | null>(null);
  const [searching, setSearching] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    const q = query.trim();
    if (q.length < 2) {
      setResults(null);
      setSearching(false);
      return;
    }
    setSearching(true);
    debounceRef.current = setTimeout(async () => {
      try {
        const found = await searchUsers(q);
        setResults(found);
      } catch {
        setResults([]);
      } finally {
        setSearching(false);
      }
    }, 250);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [query]);

  return (
    <section className="space-y-3">
      <div className="relative">
        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)]">
          <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
        </span>
        <input
          name="people-search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Find someone by name or @handle"
          className="w-full rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] py-2.5 pl-9 pr-3 text-sm outline-none placeholder:text-[var(--color-text-muted)] focus:border-[var(--color-accent)]"
        />
      </div>

      {query.trim().length >= 2 && (
        <div className="space-y-2">
          {searching && results === null ? (
            <p className="px-1 text-xs text-[var(--color-text-muted)]">
              Searching…
            </p>
          ) : results && results.length > 0 ? (
            results.map((u) => <PersonRow key={u.id} user={u} />)
          ) : (
            <p className="px-1 text-xs text-[var(--color-text-muted)]">
              No one found for &ldquo;{query.trim()}&rdquo;.
            </p>
          )}
        </div>
      )}
    </section>
  );
}

function directionCopy(direction: NetworkDirection): {
  label: string;
  tone: string;
} {
  if (direction === "you_asked") {
    return { label: "answered your ask", tone: "text-[var(--color-accent)]" };
  }
  if (direction === "connected") {
    return { label: "connected", tone: "text-[var(--color-text-secondary)]" };
  }
  if (direction === "nearby") {
    return { label: "nearby", tone: "text-[var(--color-warning)]" };
  }
  return { label: "you answered them", tone: "text-[var(--color-success)]" };
}

export default function Network() {
  usePageTitle("Network");
  const [connections, setConnections] = useState<NetworkConnection[] | null>(
    null,
  );
  const addToast = useUiStore((s) => s.addToast);

  useEffect(() => {
    let cancelled = false;
    getNetwork()
      .then((res) => {
        if (!cancelled) setConnections(res);
      })
      .catch(() => {
        if (!cancelled) {
          setConnections([]);
          addToast("Failed to load your network", "error");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [addToast]);

  if (connections === null) {
    return (
      <div className="flex justify-center py-16">
        <Spinner size="lg" />
      </div>
    );
  }

  const proximity = connections.filter(
    (c) =>
      (c.direction as NetworkDirection) === "connected" ||
      (c.direction as NetworkDirection) === "nearby",
  );
  const exchanges = connections.filter(
    (c) =>
      (c.direction as NetworkDirection) === "you_asked" ||
      (c.direction as NetworkDirection) === "you_answered",
  );
  const rows = proximity.length > 0 || exchanges.length > 0
    ? [
        { title: "Friend proximity", items: proximity },
        { title: "Perspective exchanges", items: exchanges },
      ].filter((section) => section.items.length > 0)
    : [{ title: "Your connections", items: connections }];

  return (
    <div className="space-y-6 pb-4">
      <section className="pt-4">
        <h1 className="text-[28px] font-semibold tracking-tight">Network</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          People in your orbit: those you helped, those who helped you, people
          you connected with, and close affinity matches.
        </p>
      </section>

      <PeopleSearch />

      {connections.length === 0 ? (
        <section className="space-y-3">
          <div className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border)] p-8 text-center">
            <p className="text-sm text-[var(--color-text-muted)]">
              Your network grows from moments, follows, reactions, asks, and
              answered perspectives.
            </p>
            <Link
              to="/"
              className="mt-4 inline-flex rounded-[var(--radius-sm)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
            >
              Ask something
            </Link>
          </div>
        </section>
      ) : (
        rows.map((section) => (
          <section key={section.title} className="space-y-3">
            <h2 className="text-[13px] font-semibold uppercase tracking-wide text-[var(--color-text-muted)]">
              {section.title}
            </h2>
            <div className="space-y-2">
              {section.items.map((c) => {
                const copy = directionCopy(c.direction as NetworkDirection);
                return (
                  <PersonRow
                    key={`${c.user.id}-${c.direction}`}
                    user={c.user}
                    meta={copy.label}
                    metaTone={copy.tone}
                    subtitle={c.where || c.question}
                    contextTags={c.context_tags}
                    affinity={c.affinity}
                    activeRoom={c.active_room}
                    sharedPath={c.shared_path}
                  />
                );
              })}
            </div>
          </section>
        ))
      )}
    </div>
  );
}
