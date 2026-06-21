import { useMemo, useState } from "react";
import { api } from "../api/client";
import { personHasSpouse } from "../lib/spouseFilter";
import type { Person, Relationship } from "../types";
import { displayName } from "./PersonCard";

type LinkValue = "" | `ref:${string}` | `existing:${string}`;

type BulkRow = {
  ref: string;
  given_name: string;
  patronymic: string;
  clan_name: string;
  gender: string;
  spouse: LinkValue;
  father: LinkValue;
  mother: LinkValue;
  children: LinkValue[];
};

function newRow(): BulkRow {
  return {
    ref: crypto.randomUUID(),
    given_name: "",
    patronymic: "",
    clan_name: "",
    gender: "",
    spouse: "",
    father: "",
    mother: "",
    children: [],
  };
}

function rowLabel(row: BulkRow, index: number) {
  const name = row.given_name.trim();
  return name ? `${name} (new)` : `Person ${index + 1}`;
}

function parseLink(value: LinkValue): { ref?: string; person_id?: string } | null {
  if (!value) return null;
  if (value.startsWith("ref:")) return { ref: value.slice(4) };
  if (value.startsWith("existing:")) return { person_id: value.slice(9) };
  return null;
}

function relKey(from: string, to: string, type: string) {
  return `${type}:${from}:${to}`;
}

function buildRelationships(rows: BulkRow[]) {
  const relationships: Array<{
    from: { ref?: string; person_id?: string };
    to: { ref?: string; person_id?: string };
    type: "parent" | "spouse";
  }> = [];
  const seen = new Set<string>();

  const add = (
    from: { ref?: string; person_id?: string } | null,
    to: { ref?: string; person_id?: string } | null,
    type: "parent" | "spouse",
  ) => {
    if (!from || !to) return;
    const fromKey = from.ref || from.person_id || "";
    const toKey = to.ref || to.person_id || "";
    if (fromKey === toKey) return;
    const key = relKey(fromKey, toKey, type);
    if (seen.has(key)) return;
    seen.add(key);
    relationships.push({ from, to, type });
  };

  for (const row of rows) {
    const child = { ref: row.ref };
    add(child, parseLink(row.father), "parent");
    add(child, parseLink(row.mother), "parent");

    const spouse = parseLink(row.spouse);
    if (spouse) {
      add({ ref: row.ref }, spouse, "spouse");
    }

    for (const childLink of row.children) {
      const childEndpoint = parseLink(childLink);
      add(childEndpoint, { ref: row.ref }, "parent");
    }
  }

  return relationships;
}

function LinkSelect({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: LinkValue;
  options: { value: LinkValue; label: string }[];
  onChange: (value: LinkValue) => void;
}) {
  return (
    <label className="block text-xs text-slate-600">
      <span className="mb-1 block font-medium">{label}</span>
      <select
        className="w-full rounded-lg border border-slate-200 px-2 py-1.5 text-sm"
        value={value}
        onChange={(e) => onChange(e.target.value as LinkValue)}
      >
        <option value="">—</option>
        {options.map((opt) => (
          <option key={opt.value || "none"} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    </label>
  );
}

export function BulkAddPeople({
  familyId,
  existingPeople,
  relationships,
  onAdded,
}: {
  familyId: string;
  existingPeople: Person[];
  relationships: Relationship[];
  onAdded: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [rows, setRows] = useState<BulkRow[]>([newRow(), newRow()]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const linkOptions = useMemo(() => {
    const batch = rows.map((row, index) => ({
      value: `ref:${row.ref}` as LinkValue,
      label: rowLabel(row, index),
    }));
    const existing = existingPeople.map((person) => ({
      value: `existing:${person.id}` as LinkValue,
      label: person.has_spouse
        ? `${displayName(person)} (married${person.spouse_name ? ` to ${person.spouse_name}` : ""})`
        : displayName(person),
      person,
    }));
    return [...batch, ...existing];
  }, [rows, existingPeople]);

  const spouseOptionsForRow = (rowRef: string) =>
    linkOptions
      .filter((opt) => {
        if (opt.value === `ref:${rowRef}`) return false;
        if (opt.value.startsWith("existing:")) {
          const person = "person" in opt ? opt.person : undefined;
          const personId = opt.value.slice(9);
          return !personHasSpouse(personId, relationships, person);
        }
        return true;
      })
      .map(({ value, label }) => ({ value, label }));

  const updateRow = (ref: string, patch: Partial<BulkRow>) => {
    setRows((current) => current.map((row) => (row.ref === ref ? { ...row, ...patch } : row)));
  };

  const addRow = () => setRows((current) => [...current, newRow()]);

  const removeRow = (ref: string) => {
    setRows((current) => {
      if (current.length <= 1) return current;
      return current
        .filter((row) => row.ref !== ref)
        .map((row) => ({
          ...row,
          spouse: row.spouse === `ref:${ref}` ? "" : row.spouse,
          father: row.father === `ref:${ref}` ? "" : row.father,
          mother: row.mother === `ref:${ref}` ? "" : row.mother,
          children: row.children.filter((c) => c !== `ref:${ref}`),
        }));
    });
  };

  const toggleChild = (rowRef: string, value: LinkValue) => {
    setRows((current) =>
      current.map((row) => {
        if (row.ref !== rowRef) return row;
        const has = row.children.includes(value);
        return {
          ...row,
          children: has ? row.children.filter((c) => c !== value) : [...row.children, value],
        };
      }),
    );
  };

  const submit = async () => {
    const incomplete = rows.filter(
      (row) =>
        !row.given_name.trim() &&
        (row.spouse || row.father || row.mother || row.children.length > 0),
    );
    if (incomplete.length > 0) {
      setError("Every person with relationships needs a given name.");
      return;
    }

    const people = rows.filter((row) => row.given_name.trim());
    if (people.length === 0) {
      setError("Add at least one person with a given name.");
      return;
    }

    setSaving(true);
    setError(null);
    try {
      await api.bulkCreatePeople(familyId, {
        people: people.map((row) => ({
          ref: row.ref,
          given_name: row.given_name.trim(),
          patronymic: row.patronymic.trim(),
          clan_name: row.clan_name.trim(),
          gender: row.gender,
        })),
        relationships: buildRelationships(people),
      });
      setRows([newRow(), newRow()]);
      setOpen(false);
      onAdded();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to add people";
      if (msg.includes("already has a spouse") || msg.includes("already married to")) {
        setError(`${msg}. Unlink the current spouse first, then try again.`);
      } else {
        setError(msg);
      }
    } finally {
      setSaving(false);
    }
  };

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="w-full rounded-2xl border border-dashed border-slate-300 bg-slate-50 px-4 py-4 text-left text-sm text-slate-600 transition hover:border-accent hover:bg-white hover:text-accent"
      >
        <span className="font-medium text-slate-800">Add multiple people at once</span>
        <span className="mt-1 block text-xs text-slate-500">
          Define spouses, parents, and children on one page. Spouses get their own family
          auto-created from clan or patronymic name.
        </span>
      </button>
    );
  }

  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-4">
      <div className="mb-4 flex items-start justify-between gap-3">
        <div>
          <h3 className="font-medium text-slate-900">Add multiple people</h3>
          <p className="mt-1 text-xs text-slate-500">
            Fill in each person, then pick their spouse, parents, and children from the new entries
            or people already in this family.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setOpen(false)}
          className="shrink-0 text-sm text-slate-400 hover:text-slate-600"
        >
          Close
        </button>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="space-y-4">
        {rows.map((row, index) => {
          const otherOptions = linkOptions.filter(
            (opt) => opt.value !== `ref:${row.ref}`,
          );
          const childOptions = otherOptions;

          return (
            <div key={row.ref} className="rounded-xl border border-slate-200 bg-slate-50 p-3">
              <div className="mb-3 flex items-center justify-between">
                <p className="text-sm font-medium text-slate-700">{rowLabel(row, index)}</p>
                {rows.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeRow(row.ref)}
                    className="text-xs text-slate-400 hover:text-red-600"
                  >
                    Remove
                  </button>
                )}
              </div>

              <div className="mb-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
                <input
                  className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
                  placeholder="Given name *"
                  value={row.given_name}
                  onChange={(e) => updateRow(row.ref, { given_name: e.target.value })}
                />
                <input
                  className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
                  placeholder="Patronymic"
                  value={row.patronymic}
                  onChange={(e) => updateRow(row.ref, { patronymic: e.target.value })}
                />
                <input
                  className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
                  placeholder="Family / clan name"
                  value={row.clan_name}
                  onChange={(e) => updateRow(row.ref, { clan_name: e.target.value })}
                />
                <select
                  className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
                  value={row.gender}
                  onChange={(e) => updateRow(row.ref, { gender: e.target.value })}
                >
                  <option value="">Gender</option>
                  <option value="male">Male</option>
                  <option value="female">Female</option>
                </select>
              </div>

              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                <LinkSelect
                  label="Spouse"
                  value={row.spouse}
                  options={spouseOptionsForRow(row.ref)}
                  onChange={(spouse) => updateRow(row.ref, { spouse })}
                />
                <LinkSelect
                  label="Father"
                  value={row.father}
                  options={otherOptions}
                  onChange={(father) => updateRow(row.ref, { father })}
                />
                <LinkSelect
                  label="Mother"
                  value={row.mother}
                  options={otherOptions}
                  onChange={(mother) => updateRow(row.ref, { mother })}
                />
              </div>

              <div className="mt-3">
                <p className="mb-2 text-xs font-medium text-slate-600">Children</p>
                <div className="flex flex-wrap gap-2">
                  {childOptions.length === 0 ? (
                    <span className="text-xs text-slate-400">Add more people to link children.</span>
                  ) : (
                    childOptions.map((opt) => {
                      const active = row.children.includes(opt.value);
                      return (
                        <button
                          key={opt.value}
                          type="button"
                          onClick={() => toggleChild(row.ref, opt.value)}
                          className={`rounded-full px-3 py-1 text-xs ${
                            active
                              ? "bg-accent text-white"
                              : "bg-white text-slate-600 ring-1 ring-slate-200 hover:ring-accent"
                          }`}
                        >
                          {opt.label}
                        </button>
                      );
                    })
                  )}
                </div>
              </div>
            </div>
          );
        })}
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          onClick={addRow}
          className="rounded-lg border border-slate-200 px-4 py-2 text-sm text-slate-600 hover:border-accent hover:text-accent"
        >
          + Add another person
        </button>
        <button
          type="button"
          disabled={saving}
          onClick={() => void submit()}
          className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent-hover disabled:opacity-50"
        >
          {saving ? "Saving..." : "Save all"}
        </button>
      </div>
    </div>
  );
}