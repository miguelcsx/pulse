import LoginForm from "../components/auth/LoginForm";

export default function Login() {
  return (
    <div className="min-h-dvh flex items-center justify-center bg-[var(--color-bg)] p-4">
      <div className="w-full max-w-sm">
        <div className="text-center mb-10">
          <h1 className="text-3xl font-semibold tracking-tight text-[var(--color-text)]">
            Pulse
          </h1>
          <p className="mt-1.5 text-sm text-[var(--color-text-muted)]">
            The human layer after AI
          </p>
        </div>
        <div className="rounded-[var(--radius-lg)] bg-[var(--color-bg-elevated)] border border-[var(--color-border)] p-7">
          <LoginForm />
        </div>
      </div>
    </div>
  );
}
