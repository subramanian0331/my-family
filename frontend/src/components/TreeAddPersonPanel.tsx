import { useState } from "react";
import { api } from "../api/client";
import { displayName } from "./PersonCard";
import type { Person } from "../types";

type RelationKind = "none" | "spouse" | "child" | "parent";

export function TreeAddPersonPanel({
  familyId,
  persons,
  anchorPersonId,
  variant = "floating",
  onClose,
  onAdded,
}: {
  familyId: string;
  persons: Person[];
  anchorPersonId?: string | null;
  variant?: "floating" | "inline";
  onClose: () => void;
  onAdded: () => void | Promise<void>;
}) {
  const [givenName, setGivenName] = useState("");
  const [patronymic, setPatronymic] = useState("");
  const [clanName, setClanName] = useState("");
  const [gender, setGender] = useState("");
  const [relateToId, setRelateToId] = useState(anchorPersonId ?? "");
  const [relation, setRelation] = useState<RelationKind>(anchorPersonId ? "child" : "none");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!givenName.trim()) {
      setError("Given name is required.");
      return;
    }
    if (relation !== "none" && !relateToId) {
      setError("Choose who to link this person to.");
      return;
    }

    setBusy(true);
    setError(null);
    try {
      const created = await api.createPerson(familyId, {
        given_name: givenName.trim(),
        patronymic: patronymic.trim(),
        clan_name: clanName.trim(),
        gender,
      });

      if (relation !== "none" && relateToId) {
        if (relation === "spouse") {
          await api.createRelationship(familyId, relateToId, created.id, "spouse");
        } else if (relation === "child") {
          await api.createRelationship(familyId, created.id, relateToId, "parent");
        } else if (relation === "parent") {
          await api.createRelationship(familyId, relateToId, created.id, "parent");
        }
      }

      await onAdded();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add person");
    } finally {
      setBusy(false);
    }
  };

  const anchor = anchorPersonId ? persons.find((p) => p.id === anchorPersonId) : null;

  return (
    <div
      className={`pointer-events-auto w-72 rounded-xl border border-[#c5d0dc] bg-white p-4 shadow-[0_8px_30px_rgba(30,45,60,0.16)] ${
        variant === "floating" ? "absolute right-4 top-4 z-30" : "relative mx-auto"
      }`}
      onPointerDown={(e) => e.stopPropagation()}
    >
      <div className="mb-3 flex items-start justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-[#1e2a36]">Add family member</h3>
          {anchor && (
            <p className="mt-0.5 text-xs text-[#5c6b78]">
              Linking near {displayName(anchor)}
            </p>
          )}
        </div>
        {variant === "floating" && (
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg px-2 py-1 text-sm text-[#8a8278] hover:bg-[#f3efe8]"
            aria-label="Close"
          >
            ×
          </button>
        )}
      </div>

      {error && (
        <p className="mb-3 rounded-lg bg-red-50 px-3 py-2 text-xs text-red-700">{error}</p>
      )}

      <div className="space-y-2">
        <input
          className="w-full rounded-lg border border-[#d8e0e8] px-3 py-2 text-sm outline-none focus:border-[#6fa3c8]"
          placeholder="Given name *"
          value={givenName}
          onChange={(e) => setGivenName(e.target.value)}
          autoFocus
        />
        <input
          className="w-full rounded-lg border border-[#d8e0e8] px-3 py-2 text-sm outline-none focus:border-[#6fa3c8]"
          placeholder="Patronymic"
          value={patronymic}
          onChange={(e) => setPatronymic(e.target.value)}
        />
        <input
          className="w-full rounded-lg border border-[#d8e0e8] px-3 py-2 text-sm outline-none focus:border-[#6fa3c8]"
          placeholder="Family / clan name"
          value={clanName}
          onChange={(e) => setClanName(e.target.value)}
        />
        <select
          className="w-full rounded-lg border border-[#d8e0e8] px-3 py-2 text-sm outline-none focus:border-[#6fa3c8]"
          value={gender}
          onChange={(e) => setGender(e.target.value)}
        >
          <option value="">Gender</option>
          <option value="male">Male</option>
          <option value="female">Female</option>
        </select>

        {persons.length > 0 && (
          <>
            <select
              className="w-full rounded-lg border border-[#d8e0e8] px-3 py-2 text-sm outline-none focus:border-[#6fa3c8]"
              value={relation}
              onChange={(e) => setRelation(e.target.value as RelationKind)}
            >
              <option value="none">No link yet</option>
              <option value="child">Child of…</option>
              <option value="parent">Parent of…</option>
              <option value="spouse">Spouse of…</option>
            </select>
            {relation !== "none" && (
              <select
                className="w-full rounded-lg border border-[#d8e0e8] px-3 py-2 text-sm outline-none focus:border-[#6fa3c8]"
                value={relateToId}
                onChange={(e) => setRelateToId(e.target.value)}
              >
                <option value="">Select person</option>
                {persons.map((person) => (
                  <option key={person.id} value={person.id}>
                    {displayName(person)}
                  </option>
                ))}
              </select>
            )}
          </>
        )}
      </div>

      <div className="mt-4 flex gap-2">
        <button
          type="button"
          disabled={busy}
          onClick={() => void submit()}
          className="flex-1 rounded-lg bg-[#4f86ad] px-3 py-2 text-sm font-medium text-white hover:bg-[#3d6f94] disabled:opacity-50"
        >
          {busy ? "Adding…" : "Add to tree"}
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={onClose}
          className="rounded-lg px-3 py-2 text-sm text-[#5c6b78] hover:bg-[#eef2f6]"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}