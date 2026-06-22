import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import { HomeHero } from "../components/HomeHero";
import type { FamilySummary, Invite } from "../types";

export function Families() {
  const [families, setFamilies] = useState<FamilySummary[]>([]);
  const [invites, setInvites] = useState<Invite[]>([]);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(true);
  const [scrollY, setScrollY] = useState(0);

  useEffect(() => {
    const onScroll = () => setScrollY(window.scrollY);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const load = async () => {
    try {
      const [f, i] = await Promise.all([api.families(), api.pendingInvites()]);
      setFamilies(f ?? []);
      setInvites(i ?? []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const create = async () => {
    if (!name.trim()) return;
    await api.createFamily(name.trim(), description.trim());
    setName("");
    setDescription("");
    await load();
  };

  const accept = async (token: string) => {
    await api.acceptInvite(token);
    await load();
  };

  const contentReveal = Math.min(1, scrollY / 160);

  if (loading) {
    return <div className="py-12 text-center text-brand-blue/70">Loading families...</div>;
  }

  return (
    <div className="relative -mx-4 sm:-mx-0">
      <HomeHero />

      <div
        className="relative z-10 space-y-6 rounded-t-[1.75rem] border-t border-brand-leaf/70 px-4 pb-2 pt-6 shadow-[0_-20px_56px_rgba(30,90,120,0.14)] backdrop-blur-lg sm:px-0 sm:pt-8"
        style={{
          marginTop: "-4.5rem",
          transform: `translateY(-${Math.min(scrollY * 0.12, 48)}px)`,
          backgroundColor: `rgba(244, 250, 247, ${0.82 + contentReveal * 0.18})`,
        }}
      >
        {invites.length > 0 && (
          <section className="rounded-2xl border border-amber-200/80 bg-amber-50/90 p-4 shadow-sm">
            <h2 className="mb-3 font-medium text-amber-900">Pending invites</h2>
            <div className="space-y-2">
              {invites.map((invite) => (
                <div
                  key={invite.id}
                  className="flex items-center justify-between rounded-xl bg-white/90 px-4 py-3"
                >
                  <span className="text-sm text-slate-700">Invite as {invite.role}</span>
                  <button
                    onClick={() => void accept(invite.token)}
                    className="rounded-lg bg-accent px-3 py-1.5 text-sm text-white hover:bg-accent-hover"
                  >
                    Accept
                  </button>
                </div>
              ))}
            </div>
          </section>
        )}

        <section className="rounded-2xl border border-brand-leaf bg-white/90 p-5 shadow-sm">
          <h2 className="mb-1 font-semibold text-brand-green">Create a family</h2>
          <p className="mb-4 text-sm text-brand-blue/75">Start a new tree for your relatives.</p>
          <div className="grid gap-3 sm:grid-cols-2">
            <input
              className="rounded-lg border border-brand-leaf bg-white px-3 py-2.5 outline-none focus:border-brand-teal focus:ring-2 focus:ring-brand-teal/20"
              placeholder="Family name"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <input
              className="rounded-lg border border-brand-leaf bg-white px-3 py-2.5 outline-none focus:border-brand-teal focus:ring-2 focus:ring-brand-teal/20"
              placeholder="Description (optional)"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <button
            onClick={() => void create()}
            className="mt-4 rounded-lg bg-accent px-4 py-2.5 font-medium text-white shadow-sm hover:bg-accent-hover"
          >
            Create family
          </button>
        </section>

        {families.length === 0 ? (
          <p className="rounded-2xl border border-dashed border-brand-leaf bg-white/60 px-4 py-8 text-center text-brand-blue/70">
            No families yet — create one above to get started.
          </p>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {families.map((family) => (
              <Link
                key={family.id}
                to={`/families/${family.id}`}
                className="rounded-2xl border border-brand-leaf bg-white/90 p-5 shadow-sm transition hover:border-brand-teal/50 hover:shadow-md"
              >
                <h3 className="font-semibold text-brand-green">{family.name}</h3>
                <p className="mt-1 line-clamp-2 text-sm text-brand-blue/75">
                  {family.description || "No description"}
                </p>
                <span className="mt-4 inline-block rounded-full bg-brand-leaf px-2.5 py-0.5 text-xs capitalize text-brand-green">
                  {family.role}
                </span>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}