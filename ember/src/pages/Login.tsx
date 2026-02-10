import LoginForm from "../components/auth/LoginForm";

export default function Login() {
  return (
    <div className="min-h-dvh flex items-center justify-center bg-[var(--color-bg)] p-4">
      <div className="w-full max-w-sm">
        <h1 className="text-3xl font-bold text-center mb-8">Pulse</h1>
        <LoginForm />
      </div>
    </div>
  );
}
