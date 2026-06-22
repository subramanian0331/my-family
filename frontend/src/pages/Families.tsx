import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import { CreateFamilyForm } from "../components/CreateFamilyForm";
import { Logo } from "../components/Logo";
import {
  familyInitials,
  familySubtitle,
  formatRelativeTime,
  memberCountLabel,
} from "../lib/familyDisplay";
import type { FamilySummary, Invite } from "../types";

function FamilyCard({ family }: { family: FamilySummary }) {
  const subtitle = familySubtitle(family);
  const updated = formatRelativeTime(family.updated_at);
  const meta = [memberCountLabel(family.member_count), updated].filter(Boolean).join(" · ");

  return (
    <Link
      to={`/families/${family.id}`}
      className="group flex gap-4 rounded-2xl border border-brand-leaf bg-white p-4 shadow-sm transition hover:border-brand-teal/45 hover:shadow-md sm:p-5"
    >
      <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-brand-teal/15 to-brand-blue/10 text-sm font-bold text-brand-family ring-1 ring-brand-leaf">
        {familyInitials(family.name)}
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-start justify-between gap-2">
          <h3 className="truncate font-semibold text-brand-green group-hover:text-brand-teal">
            {family.name}
          </h3>
          {family.role !== "owner" && (
            <span className="shrink-0 rounded-full bg-brand-leaf px-2 py-0.5 text-xs capitalize text-brand-green">
              {family.role}
            </span>
          )}
        </div>
        {subtitle && <p className="mt-1 line-clamp-2 text-sm text-brand-blue/75">{subtitle}</p>}
        <p className="mt-2 text-xs text-brand-blue/60">{meta}</p>
      </div>
    </Link>
  );
}

export function Families() {
  const [families, setFamilies] = useState<FamilySummary[]>([]);
  const [invites, setInvites] = useState<Invite[]>([]);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);

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
    setShowCreate(false);
    await load();
  };

  const accept = async (token: string) => {
    await api.acceptInvite(token);
    await load();
  };

  if (loading) {
    return <div className="py-12 text-center text-brand-blue/70">Loading families...</div>;
  }

  const hasFamilies = families.length > 0;

  if (!hasFamilies) {
    return (
      <div className="mx-auto max-w-lg space-y-6 py-4 text-center sm:py-8">
        <Logo variant="hero" className="mx-auto" />
        <div>
          <h1 className="font-brand text-2xl text-brand-family sm:text-3xl">
            <span className="font-medium text-brand-my">Welcome to </span>
            <span className="font-bold">My Family</span>
          </h1>
          <p className="mt-3 text-sm text-brand-blue/75 sm:text-base">
            Create your first family tree and start connecting relatives.
          </p>
        </div>

        {invites.length > 0 && (
          <section className="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-left shadow-sm">
            <h2 className="mb-3 font-medium text-amber-900">Pending invites</h2>
            <div className="space-y-2">
              {invites.map((invite) => (
                <div
                  key={invite.id}
                  className="flex items-center justify-between rounded-xl border border-amber-100 bg-white px-4 py-3"
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

        <section className="rounded-2xl border border-brand-leaf bg-white p-5 text-left shadow-sm sm:p-6">
          <h2 className="font-semibold text-brand-green">Create your first family</h2>
          <p className="mb-4 mt-1 text-sm text-brand-blue/75">Give it a name to get started.</p>
          <CreateFamilyForm
            name={name}
            description={description}
            onNameChange={setName}
            onDescriptionChange={setDescription}
            onSubmit={() => void create()}
          />
        </section>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {invites.length > 0 && (
        <section className="rounded-2xl border border-amber-200 bg-amber-50 p-4 shadow-sm">
          <h2 className="mb-3 font-medium text-amber-900">Pending invites</h2>
          <div className="space-y-2">
            {invites.map((invite) => (
              <div
                key={invite.id}
                className="flex items-center justify-between rounded-xl border border-amber-100 bg-white px-4 py-3"
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

      <section className="space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h1 className="text-2xl font-semibold text-brand-green sm:text-3xl">Your families</h1>
            <p className="mt-1 text-sm text-brand-blue/75 sm:text-base">Pick one to view the tree and search people.</p>
          </div>
          {!showCreate && (
            <button
              onClick={() => setShowCreate(true)}
              className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-accent-hover"
            >
              + New family
            </button>
          )}
        </div>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {families.map((family) => (
            <FamilyCard key={family.id} family={family} />
          ))}
        </div>
      </section>

      {showCreate ? (
        <section className="rounded-2xl border border-brand-leaf bg-white p-5 shadow-sm sm:p-6">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div>
              <h2 className="font-semibold text-brand-green">Create a family</h2>
              <p className="mt-1 text-sm text-brand-blue/75">Start a new tree for your relatives.</p>
            </div>
            <button
              onClick={() => setShowCreate(false)}
              className="rounded-lg border border-brand-leaf px-3 py-1.5 text-sm text-brand-blue/75 hover:bg-brand-mist/50"
            >
              Cancel
            </button>
          </div>
          <CreateFamilyForm
            compact
            name={name}
            description={description}
            onNameChange={setName}
            onDescriptionChange={setDescription}
            onSubmit={() => void create()}
          />
        </section>
      ) : (
        <div className="flex items-center gap-4 py-1">
          <div className="h-px flex-1 bg-brand-leaf" />
          <button
            onClick={() => setShowCreate(true)}
            className="text-sm font-medium text-brand-teal hover:text-accent-hover"
          >
            or start a new one
          </button>
          <div className="h-px flex-1 bg-brand-leaf" />
        </div>
      )}
    </div>
  );
}