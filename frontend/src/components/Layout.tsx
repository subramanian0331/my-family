import { Link, useLocation } from "react-router-dom";
import { BrandMark } from "./BrandMark";
import { HomeBackground } from "./HomeBackground";
import { useAuth } from "../context/AuthContext";

export function Layout({ children }: { children: React.ReactNode }) {
  const { user, logout } = useAuth();
  const isHome = useLocation().pathname === "/";

  return (
    <div
      className={`relative min-h-screen ${isHome ? "" : "bg-gradient-to-b from-brand-cream via-brand-mist to-[#dceaf2]"}`}
    >
      {isHome && <HomeBackground />}

      <header
        className={`sticky top-0 z-20 border-b backdrop-blur-md ${
          isHome
            ? "border-white/50 bg-white/55"
            : "border-brand-leaf/80 bg-white/85"
        }`}
      >
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-2.5 sm:py-3">
          <Link to="/" className="shrink-0 transition-opacity hover:opacity-90">
            <BrandMark />
          </Link>
          {user && (
            <div className="flex items-center gap-3">
              {user.site_role === "admin" && (
                <Link
                  to="/admin"
                  className="rounded-lg border border-violet-200/80 bg-violet-50/80 px-3 py-1.5 text-sm font-medium text-violet-700 backdrop-blur-sm hover:bg-violet-100/90"
                >
                  Admin
                </Link>
              )}
              {user.avatar_url ? (
                <img src={user.avatar_url} alt="" className="h-8 w-8 rounded-full" />
              ) : (
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-white/70 text-sm font-medium text-brand-blue">
                  {user.name?.[0] || user.email[0]}
                </div>
              )}
              <span className="hidden text-sm text-brand-blue/80 sm:inline">{user.name || user.email}</span>
              <button
                onClick={logout}
                className="rounded-lg border border-white/70 bg-white/50 px-3 py-1.5 text-sm text-brand-blue/85 backdrop-blur-sm hover:bg-white/70"
              >
                Sign out
              </button>
            </div>
          )}
        </div>
      </header>
      <main className={`relative z-10 mx-auto max-w-6xl px-4 ${isHome ? "pb-8 pt-5 sm:pt-6" : "py-6"}`}>
        {children}
      </main>
    </div>
  );
}