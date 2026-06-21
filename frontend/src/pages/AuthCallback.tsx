import { useEffect } from "react";
import { setToken } from "../api/client";

// Legacy route — backend now redirects to /?token=, but keep this as fallback.
export function AuthCallback() {
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get("token");
    if (token) {
      setToken(token);
    }
    window.location.replace("/");
  }, []);

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-50">
      <p className="text-slate-500">Signing you in...</p>
    </div>
  );
}