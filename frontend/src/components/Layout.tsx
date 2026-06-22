import { Link } from "react-router-dom";
import { Logo } from "./Logo";
import { useAuth } from "../context/AuthContext";

export function Layout({ children }: { children: React.ReactNode }) {
  const { user, logout } = useAuth();

  return (
    <div className="min-h-screen bg-gradient-to-b from-brand-cream via-brand-mist to-[#dceaf2]">
      <header className="sticky top-0 z-20 border-b border-brand-leaf/80 bg-white/85 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-2.5 sm:py-3">
          <Link to="/" className="shrink-0">
            <Logo />
          </Link>
          {user && (
            <div className="flex items-center gap-3">
              {user.site_role === "admin" && (
                <Link
                  to="/admin"
                  className="rounded-lg border border-violet-200 bg-violet-50 px-3 py-1.5 text-sm font-medium text-violet-700 hover:bg-violet-100"
                >
                  Admin
                </Link>
              )}
              {user.avatar_url ? (
                <img src={user.avatar_url} alt="" className="h-8 w-8 rounded-full" />
              ) : (
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-slate-200 text-sm font-medium">
                  {user.name?.[0] || user.email[0]}
                </div>
              )}
              <span className="hidden text-sm text-slate-600 sm:inline">{user.name || user.email}</span>
              <button
                onClick={logout}
                className="rounded-lg border border-slate-200 px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-50"
              >
                Sign out
              </button>
            </div>
          )}
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-6">{children}</main>
    </div>
  );
}