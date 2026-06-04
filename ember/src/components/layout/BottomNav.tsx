import { NavLink, useLocation } from "react-router-dom";

const navItems = [
  {
    to: "/",
    label: "Today",
    icon: (
      <svg xmlns="http://www.w3.org/2000/svg" className="h-[22px] w-[22px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="12" cy="12" r="10" />
        <polyline points="12 6 12 12 16 14" />
      </svg>
    ),
  },
  {
    to: "/commons",
    label: "Commons",
    icon: (
      <svg xmlns="http://www.w3.org/2000/svg" className="h-[22px] w-[22px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
      </svg>
    ),
  },
  {
    to: "/network",
    label: "Network",
    icon: (
      <svg xmlns="http://www.w3.org/2000/svg" className="h-[22px] w-[22px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="18" cy="5" r="3" />
        <circle cx="6" cy="12" r="3" />
        <circle cx="18" cy="19" r="3" />
        <line x1="8.6" y1="13.5" x2="15.4" y2="17.5" />
        <line x1="15.4" y1="6.5" x2="8.6" y2="10.5" />
      </svg>
    ),
  },
  {
    to: "/profile/me",
    label: "Profile",
    icon: (
      <svg xmlns="http://www.w3.org/2000/svg" className="h-[22px] w-[22px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4-4v2" />
        <circle cx="12" cy="7" r="4" />
      </svg>
    ),
  },
] as const;

export default function BottomNav() {
  const { pathname } = useLocation();

  return (
    <nav
      className="fixed bottom-0 left-0 right-0 z-40 border-t border-[var(--color-border)] bg-[var(--color-bg)]/80 backdrop-blur-xl"
      style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
    >
      <div className="mx-auto flex max-w-xl items-center justify-around py-1.5">
        {navItems.map((item) => {
          // The Profile tab routes through /profile/me but resolves to
          // /profile/:id, so rely on the path prefix instead of NavLink's
          // exact match to keep the tab highlighted while on a profile.
          const matchesProfile =
            item.label === "Profile" && pathname.startsWith("/profile");

          return (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === "/"}
              className={({ isActive }) =>
                `relative flex flex-col items-center gap-0.5 px-3 py-1 text-[10px] font-medium transition-colors ${
                  isActive || matchesProfile
                    ? "text-[var(--color-text)]"
                    : "text-[var(--color-text-muted)] active:text-[var(--color-text)]"
                }`
              }
            >
              {item.icon}
              <span>{item.label}</span>
            </NavLink>
          );
        })}
      </div>
    </nav>
  );
}
