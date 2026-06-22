import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import type { FamilySummary, Invite } from "../types";

const glassCard =
  "rounded-2xl border border-white/55 bg-white/45 p-5 shadow-[0_8px_32px_rgba(30,90,120,0.08)] backdrop-blur-md";

export function Families() {
  const [families, setFamilies] = useState<FamilySummary[]>([]);
  const [invites, setInvites] = useState<Invite[]>([]);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(true);

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

  if (loading) {
    return <div className="py-12 text-center text-brand-blue/70">Loading families...</div>;
  }

  return (
    <div className="space-y-5 sm:space-y-6">
      <p className="text-center text-sm text-brand-blue/75 sm:text-base">
        Select a family to view the tree and search people.
      </p>

      {invites.length > 0 && (
        <section className="rounded-2xl border border-amber-200/70 bg-amber-50/55 p-4 shadow-sm backdrop-blur-md">
          <h2 className="mb-3 font-medium text-amber-900">Pending invites</h2>
          <div className="space-y-2">
            {invites.map((invite) => (
              <div
                key={invite.id}
                className="flex items-center justify-between rounded-xl border border-white/50 bg-white/50 px-4 py-3 backdrop-blur-sm"
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

      <section className={glassCard}>
        <h2 className="mb-1 font-semibold text-brand-green">Create a family</h2>
        <p className="mb-4 text-sm text-brand-blue/75">Start a new tree for your relatives.</p>
        <div className="grid gap-3 sm:grid-cols-2">
          <input
            className="rounded-lg border border-white/60 bg-white/60 px-3 py-2.5 outline-none backdrop-blur-sm focus:border-brand-teal focus:ring-2 focus:ring-brand-teal/20"
            placeholder="Family name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <input
            className="rounded-lg border border-white/60 bg-white/60 px-3 py-2.5 outline-none backdrop-blur-sm focus:border-brand-teal focus:ring-2 focus:ring-brand-teal/20"
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
        <p className="rounded-2xl border border-dashed border-white/60 bg-white/35 px-4 py-8 text-center text-brand-blue/70 backdrop-blur-sm">
          No families yet — create one above to get started.
        </p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {families.map((family) => (
            <Link
              key={family.id}
              to={`/families/${family.id}`}
              className={`${glassCard} block transition hover:border-brand-teal/40 hover:bg-white/55 hover:shadow-md`}
            >
              <h3 className="font-semibold text-brand-green">{family.name}</h3>
              <p className="mt-1 line-clamp-2 text-sm text-brand-blue/75">
                {family.description || "No description"}
              </p>
              <span className="mt-4 inline-block rounded-full bg-brand-leaf/80 px-2.5 py-0.5 text-xs capitalize text-brand-green">
                {family.role}
              </span>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}