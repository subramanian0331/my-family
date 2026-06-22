import { Link, useLocation } from "react-router-dom";
import { BrandMark } from "./BrandMark";
import { HomeBackground } from "./HomeBackground";
import { useAuth } from "../context/AuthContext";

export function Layout({ children }: { children: React.ReactNode }) {
  const { user, logout } = useAuth();
  const isHome = useLocation().pathname === "/";

  return (
    <div className={`relative min-h-screen ${isHome ? "bg-brand-cream" : "bg-gradient-to-b from-brand-cream via-brand-mist to-[#dceaf2]"}`}>
      {isHome && <HomeBackground />}

      <header
        className={`sticky top-0 z-20 border-b backdrop-blur-md ${
          isHome ? "border-brand-leaf/60 bg-white/80" : "border-brand-leaf/80 bg-white/90"
        }`}
      >
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-2.5 sm:py-3">
          <Link to="/" className="shrink-0 transition-opacity hover:opacity-90">
            <BrandMark />
          </Link>
          {user && (
            <div className="flex items-center gap-2 sm:gap-3">
              {user.site_role === "admin" && (
                <Link
                  to="/admin"
                  className="rounded-lg border border-brand-teal/35 bg-white px-3 py-1.5 text-sm font-medium text-brand-teal hover:border-brand-teal/55 hover:bg-brand-mist/60"
                >
                  Admin
                </Link>
              )}
              {user.avatar_url ? (
                <img src={user.avatar_url} alt="" className="h-8 w-8 rounded-full ring-2 ring-white" />
              ) : (
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-brand-mist text-sm font-medium text-brand-blue">
                  {user.name?.[0] || user.email[0]}
                </div>
              )}
              <span className="hidden text-sm text-brand-blue/80 sm:inline">{user.name || user.email}</span>
              <button
                onClick={logout}
                className="rounded-lg border border-brand-leaf bg-white px-3 py-1.5 text-sm text-brand-blue/85 hover:border-brand-teal/40 hover:bg-brand-mist/50"
              >
                Sign out
              </button>
            </div>
          )}
        </div>
      </header>

      <main
        className={`relative z-10 mx-auto max-w-6xl px-4 ${
          isHome ? "pb-10 pt-[min(46vh,28rem)] sm:pb-12 sm:pt-[min(50vh,32rem)]" : "py-6"
        }`}
      >
        {children}
      </main>
    </div>
  );
}