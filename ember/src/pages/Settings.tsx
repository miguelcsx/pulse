import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { getMe, logout as logoutAPI, updateMe } from "../api/auth";
import { useAuthStore } from "../store/authStore";
import { useUiStore } from "../store/uiStore";
import { usePageTitle } from "../hooks/usePageTitle";
import ThemeToggle from "../components/ui/ThemeToggle";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import type { User } from "@pulse/drift/types";

export default function Settings() {
  usePageTitle("Settings");
  const navigate = useNavigate();
  const { logout, setUser } = useAuthStore();
  const addToast = useUiStore((s) => s.addToast);
  const [user, setLocalUser] = useState<User | null>(null);
  const [displayName, setDisplayName] = useState("");
  const [bio, setBio] = useState("");
  const [location, setLocation] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    getMe().then((u) => {
      setLocalUser(u);
      setDisplayName(u.display_name);
      setBio(u.bio);
      setLocation(u.location);
    });
  }, []);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      const updated = await updateMe({
        display_name: displayName,
        bio,
        location,
      });
      setUser(updated);
      setLocalUser(updated);
      addToast("Profile updated", "success");
    } catch {
      addToast("Failed to update", "error");
    } finally {
      setSaving(false);
    }
  };

  const handleLogout = async () => {
    try {
      await logoutAPI();
    } catch {
      // best-effort server-side session revocation
    }
    logout();
    navigate("/login");
  };

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold">Settings</h2>

      <div>
        <label className="block text-sm font-medium mb-2">Theme</label>
        <ThemeToggle />
      </div>

      <hr className="border-[var(--color-border)]" />

      {user && (
        <form onSubmit={handleSave} className="space-y-4">
          <Input label="Handle" value={user.handle} disabled />
          <Input label="Email" value={user.email} disabled />
          <Input
            label="Display Name"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
          <div>
            <label className="block text-sm font-medium mb-1">Bio</label>
            <textarea
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              rows={3}
              className="w-full px-3 py-2 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] text-sm text-[var(--color-text)]"
            />
          </div>
          <Input
            label="Location"
            value={location}
            onChange={(e) => setLocation(e.target.value)}
          />
          <Button type="submit" loading={saving}>
            Save Changes
          </Button>
        </form>
      )}

      <hr className="border-[var(--color-border)]" />

      <Button variant="danger" onClick={handleLogout}>
        Log Out
      </Button>
    </div>
  );
}
