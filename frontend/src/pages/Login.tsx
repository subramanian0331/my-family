import { useEffect, useState } from "react";

export function Login() {
  const [googleEnabled, setGoogleEnabled] = useState<boolean | null>(null);
  const oauthError = new URLSearchParams(window.location.search).get("error");

  useEffect(() => {
    fetch("/api/auth/status")
      .then((r) => r.json())
      .then((data: { google_enabled: boolean }) => setGoogleEnabled(data.google_enabled))
      .catch(() => setGoogleEnabled(false));
  }, []);

  return (
    <div className="flex min-h-[70vh] items-center justify-center">
      <div className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-8 text-center shadow-sm">
        <h1 className="mb-2 text-2xl font-semibold text-slate-900">Family Tree</h1>
        <p className="mb-8 text-slate-500">Build and share your family history.</p>

        {oauthError && (
          <div className="mb-6 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
            Sign-in failed ({oauthError}). Please try again.
          </div>
        )}

        {googleEnabled === false && (
          <div className="mb-6 rounded-xl border border-amber-200 bg-amber-50 p-4 text-left text-sm text-amber-900">
            <p className="font-medium">Google OAuth not configured</p>
            <p className="mt-2 text-amber-800">
              Add <code className="rounded bg-amber-100 px-1">GOOGLE_CLIENT_ID</code> and{" "}
              <code className="rounded bg-amber-100 px-1">GOOGLE_CLIENT_SECRET</code> to your{" "}
              <code className="rounded bg-amber-100 px-1">.env</code> file, then restart:
            </p>
            <pre className="mt-2 overflow-x-auto rounded bg-amber-100 p-2 text-xs">
              docker compose up --build -d
            </pre>
          </div>
        )}

        <a
          href="/api/auth/google"
          aria-disabled={googleEnabled === false}
          className={`inline-flex w-full items-center justify-center gap-2 rounded-xl px-4 py-3 font-medium text-white ${
            googleEnabled === false
              ? "pointer-events-none bg-slate-300"
              : "bg-accent hover:bg-accent-hover"
          }`}
        >
          Continue with Google
        </a>
      </div>
    </div>
  );
}