import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { api } from "../api/client";
import { BulkAddPeople } from "../components/BulkAddPeople";
import { PersonCard } from "../components/PersonCard";
import { PersonSheet } from "../components/PersonSheet";
import { TreeView } from "../components/TreeView";
import type { FamilyRole, Person, TreeData } from "../types";

export function FamilyView() {
  const { familyId = "" } = useParams();
  const [name, setName] = useState("");
  const [role, setRole] = useState<FamilyRole>("viewer");
  const [tree, setTree] = useState<TreeData>({ persons: [], relationships: [] });
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<Person[]>([]);
  const [selected, setSelected] = useState<Person | null>(null);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("viewer");
  const [newPerson, setNewPerson] = useState({
    given_name: "",
    patronymic: "",
    clan_name: "",
    gender: "",
  });
  const [tab, setTab] = useState<"tree" | "people" | "settings">("tree");
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [detail, treeData] = await Promise.all([api.family(familyId), api.tree(familyId)]);
      setName(detail.family.name);
      setRole(detail.role);
      setTree(treeData);
    } finally {
      setLoading(false);
    }
  }, [familyId]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!query.trim()) {
      setResults([]);
      return;
    }
    const timer = setTimeout(() => {
      void api.search(familyId, query).then(setResults);
    }, 250);
    return () => clearTimeout(timer);
  }, [query, familyId]);

  const addPerson = async () => {
    if (!newPerson.given_name.trim()) return;
    await api.createPerson(familyId, newPerson);
    setNewPerson({ given_name: "", patronymic: "", clan_name: "", gender: "" });
    await load();
  };

  const sendInvite = async () => {
    if (!inviteEmail.trim()) return;
    await api.createInvite(familyId, inviteEmail.trim(), inviteRole);
    setInviteEmail("");
  };

  const exportGed = async () => {
    const blob = await api.exportGedcom(familyId);
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${name || "family"}.ged`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const importGed = async (file: File) => {
    await api.importGedcom(familyId, file);
    await load();
  };

  const canEdit = role === "owner" || role === "editor";
  const canManage = role === "owner";

  const treeTabActive = tab === "tree";

  return (
    <div className={`flex flex-col ${treeTabActive ? "min-h-[calc(100dvh-5.5rem)]" : "space-y-4"}`}>
      <div className={`flex flex-wrap items-center gap-3 ${treeTabActive ? "shrink-0" : ""}`}>
        <Link to="/" className="text-sm text-slate-500 hover:text-accent">
          ← Families
        </Link>
        <h1 className="flex-1 text-2xl font-semibold">{name}</h1>
        <span className="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs capitalize text-slate-600">{role}</span>
      </div>

      <div
        className={`sticky top-[57px] z-10 rounded-xl border border-slate-200 bg-white p-3 shadow-sm ${treeTabActive ? "shrink-0" : ""}`}
      >
        <input
          className="w-full rounded-lg border border-slate-200 px-4 py-2.5 text-base outline-none focus:border-accent"
          placeholder="Search people in this family..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        {results.length > 0 && (
          <div className="mt-2 space-y-1 rounded-lg border border-slate-100 bg-slate-50 p-2">
            {results.map((person) => (
              <PersonCard key={person.id} person={person} compact onClick={() => setSelected(person)} />
            ))}
          </div>
        )}
      </div>

      <div className={`flex gap-2 border-b border-slate-200 ${treeTabActive ? "shrink-0" : ""}`}>
        {(["tree", "people", "settings"] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`border-b-2 px-4 py-2 text-sm font-medium capitalize ${
              tab === t ? "border-accent text-accent" : "border-transparent text-slate-500"
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === "tree" && (
        <div className="relative left-1/2 mt-4 flex min-h-0 w-screen max-w-none flex-1 -translate-x-1/2 flex-col px-3 sm:px-4">
          {loading ? (
            <div className="flex flex-1 items-center justify-center rounded-2xl border border-[#c5d0dc] bg-[#e8eef4] p-12 text-center text-[#5c6b78]">
              Loading tree…
            </div>
          ) : (
            <TreeView
              persons={tree.persons}
              relationships={tree.relationships}
              familyId={familyId}
              canEdit={canEdit}
              className="min-h-[calc(100dvh-15rem)]"
              onSelect={setSelected}
              onRelationshipsChanged={load}
            />
          )}
        </div>
      )}

      {tab === "people" && (
        <div className="mt-4 space-y-4">
          {canEdit && (
            <BulkAddPeople
              familyId={familyId}
              existingPeople={tree.persons}
              relationships={tree.relationships}
              onAdded={() => void load()}
            />
          )}
          {canEdit && (
            <div className="rounded-2xl border border-slate-200 bg-white p-4">
              <h3 className="mb-3 font-medium">Add person</h3>
              <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
                <input
                  className="rounded-lg border border-slate-200 px-3 py-2"
                  placeholder="Given name"
                  value={newPerson.given_name}
                  onChange={(e) => setNewPerson({ ...newPerson, given_name: e.target.value })}
                />
                <input
                  className="rounded-lg border border-slate-200 px-3 py-2"
                  placeholder="Patronymic"
                  value={newPerson.patronymic}
                  onChange={(e) => setNewPerson({ ...newPerson, patronymic: e.target.value })}
                />
                <input
                  className="rounded-lg border border-slate-200 px-3 py-2"
                  placeholder="Family / clan name"
                  value={newPerson.clan_name}
                  onChange={(e) => setNewPerson({ ...newPerson, clan_name: e.target.value })}
                />
                <select
                  className="rounded-lg border border-slate-200 px-3 py-2"
                  value={newPerson.gender}
                  onChange={(e) => setNewPerson({ ...newPerson, gender: e.target.value })}
                >
                  <option value="">Gender</option>
                  <option value="male">Male</option>
                  <option value="female">Female</option>
                </select>
              </div>
              <button
                onClick={() => void addPerson()}
                className="mt-3 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white"
              >
                Add
              </button>
            </div>
          )}
          <div className="grid gap-3 sm:grid-cols-2">
            {tree.persons.map((person) => (
              <PersonCard key={person.id} person={person} onClick={() => setSelected(person)} />
            ))}
          </div>
        </div>
      )}

      {tab === "settings" && (
        <div className="space-y-4 rounded-2xl border border-slate-200 bg-white p-5">
          {canManage && (
            <div>
              <h3 className="mb-2 font-medium">Invite someone</h3>
              <div className="flex flex-wrap gap-2">
                <input
                  className="flex-1 rounded-lg border border-slate-200 px-3 py-2"
                  placeholder="Email (must match Google account)"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                />
                <select
                  className="rounded-lg border border-slate-200 px-3 py-2"
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value)}
                >
                  <option value="viewer">Viewer</option>
                  <option value="editor">Editor</option>
                </select>
                <button
                  onClick={() => void sendInvite()}
                  className="rounded-lg bg-accent px-4 py-2 text-white"
                >
                  Send invite
                </button>
              </div>
            </div>
          )}
          {canEdit && (
            <div className="flex flex-wrap gap-3">
              <button onClick={() => void exportGed()} className="rounded-lg border border-slate-200 px-4 py-2 text-sm">
                Export GEDCOM
              </button>
              <label className="cursor-pointer rounded-lg border border-slate-200 px-4 py-2 text-sm">
                Import GEDCOM
                <input
                  type="file"
                  accept=".ged,.GED"
                  className="hidden"
                  onChange={(e) => {
                    const file = e.target.files?.[0];
                    if (file) void importGed(file);
                  }}
                />
              </label>
            </div>
          )}
        </div>
      )}

      {selected && (
        <PersonSheet
          key={selected.id}
          person={selected}
          familyId={familyId}
          role={role}
          onClose={() => setSelected(null)}
          onUpdated={async () => {
            const treeData = await api.tree(familyId);
            setTree(treeData);
            const updated = treeData.persons.find((p) => p.id === selected.id);
            if (updated) setSelected(updated);
          }}
        />
      )}
    </div>
  );
}