import LoginForm from "../components/auth/LoginForm";

export default function Login() {
  return (
    <div className="min-h-dvh flex items-center justify-center bg-[var(--color-bg)] p-4">
      <div className="w-full max-w-sm">
        <h1 className="text-3xl font-bold text-center mb-2 bg-gradient-to-r from-[var(--color-primary)] to-pink-500 bg-clip-text text-transparent">
          Pulse
        </h1>
        <p className="text-center text-sm text-[var(--color-text-muted)] mb-8">
          Connect through moments
        </p>
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-2xl p-8">
          <LoginForm />
        </div>
      </div>
    </div>
  );
}
