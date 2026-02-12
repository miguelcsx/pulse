import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import type { AxiosError } from "axios";
import client from "../../api/client";
import { useAuthStore } from "../../store/authStore";
import type { AuthTokens } from "@pulse/drift/types";
import Input from "../ui/Input";
import Button from "../ui/Button";

export default function RegisterForm() {
  const [handle, setHandle] = useState("");
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const setTokens = useAuthStore((s) => s.setTokens);
  const setUser = useAuthStore((s) => s.setUser);
  const navigate = useNavigate();

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await client.post<AuthTokens>("/auth/register", {
        handle,
        email,
        password,
        display_name: displayName,
      });
      setTokens(res.data.access_token);
      setUser(res.data.user);
      navigate("/");
    } catch (err) {
      const axiosErr = err as AxiosError<{ error: string }>;
      setError(axiosErr.response?.data?.error ?? "Registration failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <Input
        label="Handle"
        type="text"
        value={handle}
        onChange={(e) => setHandle(e.target.value)}
        placeholder="your_handle"
        required
      />
      <Input
        label="Email"
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="you@example.com"
        required
      />
      <Input
        label="Display Name"
        type="text"
        value={displayName}
        onChange={(e) => setDisplayName(e.target.value)}
        placeholder="Display Name"
        required
      />
      <Input
        label="Password"
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        placeholder="Password"
        required
      />

      {error && (
        <p className="text-sm text-[var(--color-error)]">{error}</p>
      )}

      <Button type="submit" loading={loading}>
        Create account
      </Button>

      <p className="text-center text-sm text-[var(--color-text-muted)]">
        Already have an account?{" "}
        <Link to="/login" className="text-[var(--color-primary)] hover:opacity-80">
          Sign in
        </Link>
      </p>
    </form>
  );
}
