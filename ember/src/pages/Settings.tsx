import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { getMe, logout as logoutAPI, updateMe } from "../api/auth";
import { getToday, updateTrustProfile } from "../api/advice";
import { useAuthStore } from "../store/authStore";
import { useUiStore } from "../store/uiStore";
import { usePageTitle } from "../hooks/usePageTitle";
import ThemeToggle from "../components/ui/ThemeToggle";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import type { Availability, User } from "@pulse/drift/types";

export default function Settings() {
  usePageTitle("Settings");
  const navigate = useNavigate();
  const { logout, setUser } = useAuthStore();
  const addToast = useUiStore((s) => s.addToast);
  const [user, setLocalUser] = useState<User | null>(null);
  const [displayName, setDisplayName] = useState("");
  const [bio, setBio] = useState("");
  const [location, setLocation] = useState("");
  const [topics, setTopics] = useState("");
  const [livedExperience, setLivedExperience] = useState("");
  const [availability, setAvailability] = useState<Availability>("async");
  const [saving, setSaving] = useState(false);
  const [savingTrust, setSavingTrust] = useState(false);

  useEffect(() => {
    getMe().then((u) => {
      setLocalUser(u);
      setDisplayName(u.display_name);
      setBio(u.bio);
      setLocation(u.location);
    });
    getToday().then((today) => {
      if (today.trust_profile) {
        setTopics(today.trust_profile.topics);
        setLivedExperience(today.trust_profile.lived_experience);
        setAvailability(today.trust_profile.availability);
      }
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

  const handleTrustSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSavingTrust(true);
    try {
      await updateTrustProfile({
        topics,
        lived_experience: livedExperience,
        availability,
      });
      addToast("Trust profile updated", "success");
    } catch {
      addToast("Failed to update trust profile", "error");
    } finally {
      setSavingTrust(false);
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
            <label htmlFor="profile-bio" className="block text-sm font-medium mb-1">
              Bio
            </label>
            <textarea
              id="profile-bio"
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

      <form onSubmit={handleTrustSave} className="space-y-4">
        <div>
          <h3 className="text-sm font-semibold">Trust profile</h3>
          <p className="mt-1 text-xs text-[var(--color-text-muted)]">
            Tell Pulse where you can help as a mentor or peer.
          </p>
        </div>
        <div>
          <label htmlFor="trust-topics" className="block text-sm font-medium mb-1">
            Topics
          </label>
          <textarea
            id="trust-topics"
            value={topics}
            onChange={(e) => setTopics(e.target.value)}
            rows={3}
            placeholder="first customers, creative direction, portfolio review"
            className="w-full px-3 py-2 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] text-sm text-[var(--color-text)]"
          />
        </div>
        <div>
          <label htmlFor="trust-lived-experience" className="block text-sm font-medium mb-1">
            Lived experience
          </label>
          <textarea
            id="trust-lived-experience"
            value={livedExperience}
            onChange={(e) => setLivedExperience(e.target.value)}
            rows={5}
            placeholder="What have you lived through that could help someone else?"
            className="w-full px-3 py-2 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] text-sm text-[var(--color-text)]"
          />
        </div>
        <div>
          <label htmlFor="trust-availability" className="block text-sm font-medium mb-1">
            Availability
          </label>
          <select
            id="trust-availability"
            value={availability}
            onChange={(e) => setAvailability(e.target.value as Availability)}
            className="w-full px-3 py-2 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] text-sm text-[var(--color-text)]"
          >
            <option value="async">Async</option>
            <option value="live_now">Live now</option>
            <option value="bookable_10m">Bookable 10m</option>
          </select>
        </div>
        <Button type="submit" loading={savingTrust}>
          Save Trust Profile
        </Button>
      </form>

      <hr className="border-[var(--color-border)]" />

      <Button variant="danger" onClick={handleLogout}>
        Log Out
      </Button>
    </div>
  );
}
