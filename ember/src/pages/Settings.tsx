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

function Section({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border)] p-5 space-y-4">
      <div>
        <h3 className="text-[15px] font-semibold">{title}</h3>
        {subtitle && (
          <p className="mt-0.5 text-xs text-[var(--color-text-muted)]">
            {subtitle}
          </p>
        )}
      </div>
      {children}
    </section>
  );
}

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
      // best-effort
    }
    logout();
    navigate("/login");
  };

  return (
    <div className="space-y-5 pb-4">
      <h2 className="text-[22px] font-semibold tracking-tight pt-2">
        Settings
      </h2>

      {/* Appearance */}
      <Section title="Appearance">
        <ThemeToggle />
      </Section>

      {/* Profile */}
      {user && (
        <Section title="Profile" subtitle="Your public information">
          <form onSubmit={handleSave} className="space-y-4">
            <Input label="Handle" value={user.handle} disabled />
            <Input label="Email" value={user.email} disabled />
            <Input
              label="Display Name"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
            <div>
              <label
                htmlFor="profile-bio"
                className="block text-[13px] font-medium text-[var(--color-text-secondary)] mb-1.5"
              >
                Bio
              </label>
              <textarea
                id="profile-bio"
                value={bio}
                onChange={(e) => setBio(e.target.value)}
                rows={3}
                className="w-full px-3 py-2.5 rounded-[var(--radius-sm)] bg-[var(--color-bg)] border border-[var(--color-border)] text-sm text-[var(--color-text)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent"
              />
            </div>
            <Input
              label="Location"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
            />
            <Button type="submit" loading={saving}>
              Save
            </Button>
          </form>
        </Section>
      )}

      {/* Trust profile */}
      <Section
        title="Trust profile"
        subtitle="How Pulse matches you with people who need help"
      >
        <form onSubmit={handleTrustSave} className="space-y-4">
          <div>
            <label
              htmlFor="trust-topics"
              className="block text-[13px] font-medium text-[var(--color-text-secondary)] mb-1.5"
            >
              Topics you can help with
            </label>
            <textarea
              id="trust-topics"
              value={topics}
              onChange={(e) => setTopics(e.target.value)}
              rows={2}
              placeholder="first customers, creative direction, portfolio review"
              className="w-full px-3 py-2.5 rounded-[var(--radius-sm)] bg-[var(--color-bg)] border border-[var(--color-border)] text-sm text-[var(--color-text)] placeholder-[var(--color-text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent"
            />
          </div>
          <div>
            <label
              htmlFor="trust-lived-experience"
              className="block text-[13px] font-medium text-[var(--color-text-secondary)] mb-1.5"
            >
              Lived experience
            </label>
            <textarea
              id="trust-lived-experience"
              value={livedExperience}
              onChange={(e) => setLivedExperience(e.target.value)}
              rows={4}
              placeholder="What have you lived through that could help someone else?"
              className="w-full px-3 py-2.5 rounded-[var(--radius-sm)] bg-[var(--color-bg)] border border-[var(--color-border)] text-sm text-[var(--color-text)] placeholder-[var(--color-text-muted)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent"
            />
          </div>
          <div>
            <label
              htmlFor="trust-availability"
              className="block text-[13px] font-medium text-[var(--color-text-secondary)] mb-1.5"
            >
              Availability
            </label>
            <select
              id="trust-availability"
              value={availability}
              onChange={(e) =>
                setAvailability(e.target.value as Availability)
              }
              className="w-full px-3 py-2.5 rounded-[var(--radius-sm)] bg-[var(--color-bg)] border border-[var(--color-border)] text-sm text-[var(--color-text)] focus:outline-none focus:ring-2 focus:ring-[var(--color-accent)] focus:border-transparent"
            >
              <option value="async">Async</option>
              <option value="live_now">Live now</option>
              <option value="bookable_10m">Bookable 10m</option>
            </select>
          </div>
          <Button type="submit" loading={savingTrust}>
            Save
          </Button>
        </form>
      </Section>

      {/* Logout */}
      <Section title="Account">
        <Button variant="danger" onClick={handleLogout} className="w-full">
          Log out
        </Button>
      </Section>
    </div>
  );
}
