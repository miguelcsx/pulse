import { useUiStore } from "../../store/uiStore";

const typeClasses = {
  success: "border-[var(--color-success)] bg-[var(--color-surface)] text-[var(--color-success)]",
  error: "border-[var(--color-error)] bg-[var(--color-surface)] text-[var(--color-error)]",
  info: "border-[var(--color-primary)] bg-[var(--color-surface)] text-[var(--color-primary)]",
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
          className={`flex items-center gap-3 rounded-lg border px-4 py-3 text-sm shadow-lg ${typeClasses[toast.type]}`}
        >
          <span className="flex-1">{toast.message}</span>
          <button
            onClick={() => removeToast(toast.id)}
            className="shrink-0 opacity-70 hover:opacity-100 transition-opacity"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-4 w-4"
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
