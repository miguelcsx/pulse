import { useUiStore } from "../../store/uiStore";

const typeClasses = {
  success:
    "bg-[var(--color-bg-elevated)] border-[var(--color-success)] text-[var(--color-text)]",
  error:
    "bg-[var(--color-bg-elevated)] border-[var(--color-error)] text-[var(--color-text)]",
  info: "bg-[var(--color-bg-elevated)] border-[var(--color-accent)] text-[var(--color-text)]",
} as const;

const dotClasses = {
  success: "bg-[var(--color-success)]",
  error: "bg-[var(--color-error)]",
  info: "bg-[var(--color-accent)]",
} as const;

export default function Toast() {
  const toasts = useUiStore((s) => s.toasts);
  const removeToast = useUiStore((s) => s.removeToast);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed top-4 right-4 z-50 flex flex-col gap-2">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={`flex items-center gap-2.5 rounded-[var(--radius-sm)] border px-4 py-3 text-sm shadow-lg animate-slide-up ${typeClasses[toast.type]}`}
        >
          <span
            className={`h-1.5 w-1.5 shrink-0 rounded-full ${dotClasses[toast.type]}`}
          />
          <span className="flex-1">{toast.message}</span>
          <button
            onClick={() => removeToast(toast.id)}
            className="shrink-0 opacity-50 hover:opacity-100 transition-opacity"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-3.5 w-3.5"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
      ))}
    </div>
  );
}
