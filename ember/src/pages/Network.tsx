import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import type { NetworkConnection, User } from "@pulse/drift/types";
import { getNetwork } from "../api/advice";
import { searchUsers } from "../api/social";
import Spinner from "../components/ui/Spinner";
import { usePageTitle } from "../hooks/usePageTitle";
import { useUiStore } from "../store/uiStore";

function PersonRow({
  user,
  meta,
  metaTone,
  subtitle,
}: {
  user: Pick<User, "id" | "handle" | "display_name">;
  meta?: string;
  metaTone?: string;
  subtitle?: string;
}) {
  return (
    <Link
      to={`/profile/${user.id}`}
      className="flex items-center gap-3 rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-bg-elevated)] p-4 transition-colors hover:bg-[var(--color-surface)]"
    >
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

function directionCopy(direction: NetworkConnection["direction"]): {
  label: string;
  tone: string;
} {
  if (direction === "you_asked") {
    return { label: "answered your ask", tone: "text-[var(--color-accent)]" };
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

  return (
    <div className="space-y-6 pb-4">
      <section className="pt-4">
        <h1 className="text-[28px] font-semibold tracking-tight">Network</h1>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Find anyone by handle, or revisit the people you&rsquo;ve exchanged
          perspective with. No followers, no vanity — only real conversations.
        </p>
      </section>

      <PeopleSearch />

      <section className="space-y-3">
        <h2 className="text-[13px] font-semibold uppercase tracking-wide text-[var(--color-text-muted)]">
          Your connections
        </h2>
        {connections.length === 0 ? (
          <div className="rounded-[var(--radius-md)] border border-dashed border-[var(--color-border)] p-8 text-center">
            <p className="text-sm text-[var(--color-text-muted)]">
              Your network grows as people answer your asks — and as you answer
              theirs.
            </p>
            <Link
              to="/"
              className="mt-4 inline-flex rounded-[var(--radius-sm)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
            >
              Ask something
            </Link>
          </div>
        ) : (
          <div className="space-y-2">
            {connections.map((c) => {
              const copy = directionCopy(c.direction);
              return (
                <PersonRow
                  key={`${c.user.id}-${c.direction}`}
                  user={c.user}
                  meta={copy.label}
                  metaTone={copy.tone}
                  subtitle={c.question}
                />
              );
            })}
          </div>
        )}
      </section>
    </div>
  );
}
