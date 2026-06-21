import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "../api/client";
import type {
  FamilyRole,
  FamilySummary,
  Person,
  PersonFamilyRef,
  PersonSearchHit,
  Relationship,
} from "../types";
import {
  isActiveSpouseRelationship,
  maritalStatus,
  relationshipsForPerson,
} from "../lib/relationshipMetadata";
import { isAvailableSpousePartner } from "../lib/spouseFilter";
import { isDeceased, lifeSpan } from "../lib/personUtils";
import { displayName } from "./PersonCard";
import { PersonPhoto } from "./PersonPhoto";

type LinkedRelation = {
  relationshipId: string;
  personId: string;
  person: Person | null;
  divorced?: boolean;
};

function relationLabel(link: LinkedRelation) {
  return link.person ? displayName(link.person) : "Unknown member";
}

export function PersonSheet({
  person,
  familyId,
  role,
  onClose,
  onUpdated,
}: {
  person: Person;
  familyId: string;
  role: FamilyRole;
  onClose: () => void;
  onUpdated: () => void | Promise<void>;
}) {
  const canEdit = role === "owner" || role === "editor";
  const [form, setForm] = useState(person);
  const [allPeople, setAllPeople] = useState<Person[]>([]);
  const [relationships, setRelationships] = useState<Relationship[]>([]);
  const [loadingPeople, setLoadingPeople] = useState(true);
  const [peopleLoadError, setPeopleLoadError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [linkError, setLinkError] = useState<string | null>(null);
  const [showNewParent, setShowNewParent] = useState(false);
  const [newParent, setNewParent] = useState({ given_name: "", patronymic: "", gender: "male" });
  const [linkMode, setLinkMode] = useState<"parent" | "child" | "spouse">("parent");
  const [unlinkingId, setUnlinkingId] = useState<string | null>(null);
  const [divorcingId, setDivorcingId] = useState<string | null>(null);
  const [linkSource, setLinkSource] = useState<"family" | "all">("family");
  const [globalQuery, setGlobalQuery] = useState("");
  const [globalResults, setGlobalResults] = useState<PersonSearchHit[]>([]);
  const [globalSearching, setGlobalSearching] = useState(false);
  const [personFamilies, setPersonFamilies] = useState<PersonFamilyRef[]>([]);
  const [userFamilies, setUserFamilies] = useState<FamilySummary[]>([]);
  const [familiesLoading, setFamiliesLoading] = useState(true);
  const [familyActionError, setFamilyActionError] = useState<string | null>(null);
  const [removingFamilyId, setRemovingFamilyId] = useState<string | null>(null);
  const [addingFamilyId, setAddingFamilyId] = useState<string | null>(null);
  const [togglingFamilyId, setTogglingFamilyId] = useState<string | null>(null);

  useEffect(() => {
    setForm({
      ...person,
      deceased: Boolean(person.deceased || person.death_date),
    });
  }, [person]);

  const refreshTreeData = useCallback(async () => {
    const data = await api.tree(familyId);
    setAllPeople(data.persons);
    setRelationships(data.relationships);
    return data;
  }, [familyId]);

  const notifyUpdated = useCallback(async () => {
    await refreshTreeData();
    await onUpdated();
  }, [refreshTreeData, onUpdated]);

  useEffect(() => {
    setLoadingPeople(true);
    setPeopleLoadError(null);
    void refreshTreeData()
      .catch((err) => {
        setAllPeople([]);
        setRelationships([]);
        setPeopleLoadError(err instanceof Error ? err.message : "Failed to load family members");
      })
      .finally(() => setLoadingPeople(false));
  }, [person.id, refreshTreeData]);

  useEffect(() => {
    setFamiliesLoading(true);
    setFamilyActionError(null);
    void Promise.all([api.personFamilies(person.id), api.families()])
      .then(([assigned, all]) => {
        setPersonFamilies(assigned);
        setUserFamilies(all);
      })
      .catch((err) => {
        setPersonFamilies([]);
        setUserFamilies([]);
        setFamilyActionError(err instanceof Error ? err.message : "Failed to load families");
      })
      .finally(() => setFamiliesLoading(false));
  }, [person.id]);

  useEffect(() => {
    if (linkSource !== "all") {
      setGlobalResults([]);
      return;
    }
    const q = globalQuery.trim();
    if (!q) {
      setGlobalResults([]);
      return;
    }
    setGlobalSearching(true);
    const timer = setTimeout(() => {
      void api
        .searchPeople(q, familyId)
        .then(setGlobalResults)
        .catch(() => setGlobalResults([]))
        .finally(() => setGlobalSearching(false));
    }, 250);
    return () => clearTimeout(timer);
  }, [globalQuery, linkSource, familyId]);

  const byId = useMemo(() => new Map(allPeople.map((p) => [p.id, p])), [allPeople]);
  const memberIds = useMemo(() => new Set(allPeople.map((p) => p.id)), [allPeople]);
  const personRelationships = useMemo(
    () => relationshipsForPerson(person.id, relationships, memberIds),
    [person.id, relationships, memberIds],
  );

  const parentLinks = useMemo(
    () =>
      personRelationships
        .filter((r) => r.type === "parent" && r.from_person_id === person.id)
        .map((r) => ({
          relationshipId: r.id,
          personId: r.to_person_id,
          person: byId.get(r.to_person_id) ?? null,
        })),
    [personRelationships, person.id, byId],
  );

  const childLinks = useMemo(
    () =>
      personRelationships
        .filter((r) => r.type === "parent" && r.to_person_id === person.id)
        .map((r) => ({
          relationshipId: r.id,
          personId: r.from_person_id,
          person: byId.get(r.from_person_id) ?? null,
        })),
    [personRelationships, person.id, byId],
  );

  const spouseLinks = useMemo(() => {
    const seen = new Set<string>();
    const result: LinkedRelation[] = [];
    for (const rel of personRelationships) {
      if (rel.type !== "spouse") continue;
      const otherId =
        rel.from_person_id === person.id ? rel.to_person_id : rel.from_person_id;
      if (otherId === person.id || seen.has(otherId)) continue;
      seen.add(otherId);
      result.push({
        relationshipId: rel.id,
        personId: otherId,
        person: byId.get(otherId) ?? null,
        divorced: maritalStatus(rel) === "divorced",
      });
    }
    return result;
  }, [personRelationships, person.id, byId]);

  const activeSpouseLinks = useMemo(
    () => spouseLinks.filter((link) => !link.divorced),
    [spouseLinks],
  );
  const divorcedSpouseLinks = useMemo(
    () => spouseLinks.filter((link) => link.divorced),
    [spouseLinks],
  );

  const linkedParentIds = useMemo(
    () =>
      new Set(
        personRelationships
          .filter((r) => r.type === "parent" && r.from_person_id === person.id)
          .map((r) => r.to_person_id),
      ),
    [personRelationships, person.id],
  );
  const linkedChildIds = useMemo(
    () =>
      new Set(
        personRelationships
          .filter((r) => r.type === "parent" && r.to_person_id === person.id)
          .map((r) => r.from_person_id),
      ),
    [personRelationships, person.id],
  );
  const linkedSpouseIds = useMemo(() => {
    const ids = new Set<string>();
    for (const rel of personRelationships) {
      if (!isActiveSpouseRelationship(rel)) continue;
      const otherId =
        rel.from_person_id === person.id ? rel.to_person_id : rel.from_person_id;
      if (otherId !== person.id) ids.add(otherId);
    }
    return ids;
  }, [personRelationships, person.id]);

  const currentPersonRecord = useMemo(
    () => allPeople.find((p) => p.id === person.id) ?? person,
    [allPeople, person],
  );

  const isLinkable = (id: string, candidate?: Pick<Person, "has_spouse">) => {
    if (id === person.id) return false;
    switch (linkMode) {
      case "parent":
        return !linkedParentIds.has(id);
      case "child":
        return !linkedChildIds.has(id);
      case "spouse":
        return isAvailableSpousePartner(id, person.id, personRelationships, candidate);
    }
  };

  const linkableForMode = useMemo(
    () => allPeople.filter((p) => isLinkable(p.id, p)),
    [allPeople, person.id, linkMode, linkedParentIds, linkedChildIds, personRelationships, currentPersonRecord],
  );

  const linkableGlobalResults = useMemo(
    () =>
      globalResults.filter(
        (hit) => isLinkable(hit.person.id, hit.person) && !hit.in_target_family,
      ),
    [globalResults, person.id, linkMode, linkedParentIds, linkedChildIds, personRelationships, currentPersonRecord],
  );

  const assignedFamilyIds = useMemo(
    () => new Set(personFamilies.map((f) => f.id)),
    [personFamilies],
  );

  const addableFamilies = useMemo(
    () =>
      userFamilies.filter(
        (f) =>
          (f.role === "owner" || f.role === "editor") && !assignedFamilyIds.has(f.id),
      ),
    [userFamilies, assignedFamilyIds],
  );

  const refreshFamilies = async () => {
    const assigned = await api.personFamilies(person.id);
    setPersonFamilies(assigned);
  };

  const setMarriedInForFamily = async (targetFamilyId: string, marriedIn: boolean) => {
    setFamilyActionError(null);
    setTogglingFamilyId(targetFamilyId);
    try {
      const assigned = await api.setFamilyMarriageLabel(targetFamilyId, person.id, marriedIn);
      setPersonFamilies(assigned);
      if (targetFamilyId === familyId) {
        await notifyUpdated();
      }
    } catch (err) {
      setFamilyActionError(err instanceof Error ? err.message : "Failed to update family label");
    } finally {
      setTogglingFamilyId(null);
    }
  };

  const addToFamily = async (targetFamilyId: string) => {
    setFamilyActionError(null);
    setAddingFamilyId(targetFamilyId);
    try {
      await api.addPersonToFamily(targetFamilyId, person.id);
      await refreshFamilies();
      if (targetFamilyId === familyId) {
        await notifyUpdated();
      }
    } catch (err) {
      setFamilyActionError(err instanceof Error ? err.message : "Failed to add family");
    } finally {
      setAddingFamilyId(null);
    }
  };

  const removeFromFamily = async (targetFamilyId: string) => {
    setFamilyActionError(null);
    setRemovingFamilyId(targetFamilyId);
    try {
      await api.removePersonFromFamily(targetFamilyId, person.id);
      await refreshFamilies();
      if (targetFamilyId === familyId) {
        await notifyUpdated();
        onClose();
      }
    } catch (err) {
      setFamilyActionError(err instanceof Error ? err.message : "Failed to remove family");
    } finally {
      setRemovingFamilyId(null);
    }
  };

  const save = async () => {
    setSaving(true);
    setSaveError(null);
    try {
      await api.updatePerson(person.id, familyId, {
        ...form,
        deceased: Boolean(form.deceased),
        birth_date: form.birth_date || undefined,
        death_date: form.deceased ? form.death_date || undefined : undefined,
        death_place: form.deceased ? form.death_place : "",
      });
      await notifyUpdated();
      onClose();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Failed to save");
    } finally {
      setSaving(false);
    }
  };

  const suggestPatronymic = async () => {
    const response = await fetch(`/api/people/${person.id}/patronymic-suggestion`, {
      headers: { Authorization: `Bearer ${localStorage.getItem("family_tree_token")}` },
    });
    const data = await response.json();
    setForm((f) => ({ ...f, patronymic: data.patronymic || f.patronymic }));
  };

  const uploadPhoto = async (file: File) => {
    await api.uploadPhoto(person.id, familyId, file);
    await notifyUpdated();
  };

  const ensureInFamily = async (otherId: string, inTargetFamily?: boolean) => {
    if (inTargetFamily || allPeople.some((p) => p.id === otherId)) return;
    await api.addPersonToFamily(familyId, otherId);
  };

  const formatLinkError = (err: unknown, fallback: string) => {
    const msg = err instanceof Error ? err.message : fallback;
    if (msg.includes("already has a spouse") || msg.includes("already married to")) {
      return `${msg}. Unlink the current spouse first, then try again.`;
    }
    return msg;
  };

  const linkRelationship = async (
    otherId: string,
    type: "parent" | "child" | "spouse",
    inTargetFamily?: boolean,
  ) => {
    setLinkError(null);
    try {
      // Spouse labels are applied by the API; pre-adding runs EnsureNativeFamily and
      // labels the partner as native, which used to cascade into other families.
      if (type !== "spouse") {
        await ensureInFamily(otherId, inTargetFamily);
      }
      if (type === "child") {
        await api.createRelationship(familyId, otherId, person.id, "parent");
      } else if (type === "parent") {
        await api.createRelationship(familyId, person.id, otherId, "parent");
      } else {
        await api.createRelationship(familyId, person.id, otherId, "spouse");
      }
      await refreshFamilies();
      setGlobalQuery("");
      setGlobalResults([]);
      await notifyUpdated();
    } catch (err) {
      setLinkError(formatLinkError(err, "Failed to link relationship"));
    }
  };

  const markDivorced = async (relationshipId: string) => {
    setLinkError(null);
    setDivorcingId(relationshipId);
    try {
      await api.updateRelationship(relationshipId, familyId, { marital_status: "divorced" });
      await notifyUpdated();
    } catch (err) {
      setLinkError(err instanceof Error ? err.message : "Failed to mark as divorced");
    } finally {
      setDivorcingId(null);
    }
  };

  const unlinkRelationship = async (relationshipId: string) => {
    setLinkError(null);
    setUnlinkingId(relationshipId);
    try {
      await api.deleteRelationship(relationshipId, familyId);
      await notifyUpdated();
    } catch (err) {
      setLinkError(err instanceof Error ? err.message : "Failed to unlink relationship");
    } finally {
      setUnlinkingId(null);
    }
  };

  const createAndLinkParent = async () => {
    if (!newParent.given_name.trim()) {
      setLinkError("Parent given name is required");
      return;
    }
    setLinkError(null);
    try {
      const created = await api.createPerson(familyId, newParent);
      await api.createRelationship(familyId, person.id, created.id, "parent");
      setNewParent({ given_name: "", patronymic: "", gender: "male" });
      setShowNewParent(false);
      await notifyUpdated();
    } catch (err) {
      setLinkError(err instanceof Error ? err.message : "Failed to create parent");
    }
  };

  const emptyLinkMessage = () => {
    if (loadingPeople) return "Loading family members...";
    if (peopleLoadError) return peopleLoadError;
    if (allPeople.length <= 1) {
      return "No other people in this family yet. Use “Create new parent” below.";
    }
    if (linkableForMode.length === 0) {
      if (linkMode === "parent") {
        return "All other members are already linked as parents. Try “As child” or “As spouse”, or create a new parent.";
      }
      if (linkMode === "child") return "All other members are already linked as children.";
      if (currentPersonRecord.has_spouse || linkedSpouseIds.size > 0) {
        return "Unlink the current spouse before linking a new one.";
      }
      return "No unmarried members available — everyone listed already has a spouse.";
    }
    return null;
  };

  return (
    <div className="fixed inset-0 z-40 flex items-end justify-center bg-black/30 sm:items-center">
      <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-t-2xl bg-white p-6 shadow-xl sm:rounded-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-xl font-semibold">{displayName(currentPersonRecord)}</h2>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-600">
            Close
          </button>
        </div>

        {currentPersonRecord.photo_id && (
          <PersonPhoto
            photoId={currentPersonRecord.photo_id}
            alt=""
            className="mb-4 h-32 w-32 rounded-2xl object-cover"
          />
        )}

        {(parentLinks.length > 0 || childLinks.length > 0 || spouseLinks.length > 0) && (
          <div className="mb-4 space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-3">
            {parentLinks.length > 0 && (
              <div>
                <p className="mb-1 text-xs font-medium uppercase tracking-wide text-slate-500">
                  Parents
                </p>
                <div className="flex flex-wrap gap-2">
                  {parentLinks.map((link) => (
                    <RelationChip
                      key={link.relationshipId}
                      link={link}
                      canEdit={canEdit}
                      unlinking={unlinkingId === link.relationshipId}
                      onUnlink={() => void unlinkRelationship(link.relationshipId)}
                    />
                  ))}
                </div>
              </div>
            )}
            {activeSpouseLinks.length > 0 && (
              <div>
                <p className="mb-1 text-xs font-medium uppercase tracking-wide text-slate-500">
                  Spouse
                </p>
                <div className="flex flex-wrap gap-2">
                  {activeSpouseLinks.map((link) => (
                    <RelationChip
                      key={link.relationshipId}
                      link={link}
                      canEdit={canEdit}
                      unlinking={unlinkingId === link.relationshipId}
                      divorcing={divorcingId === link.relationshipId}
                      onDivorce={() => void markDivorced(link.relationshipId)}
                      onUnlink={() => void unlinkRelationship(link.relationshipId)}
                    />
                  ))}
                </div>
              </div>
            )}
            {divorcedSpouseLinks.length > 0 && (
              <div>
                <p className="mb-1 text-xs font-medium uppercase tracking-wide text-slate-500">
                  Former spouses
                </p>
                <div className="flex flex-wrap gap-2">
                  {divorcedSpouseLinks.map((link) => (
                    <RelationChip
                      key={link.relationshipId}
                      link={link}
                      canEdit={canEdit}
                      unlinking={unlinkingId === link.relationshipId}
                      onUnlink={() => void unlinkRelationship(link.relationshipId)}
                    />
                  ))}
                </div>
              </div>
            )}
            {childLinks.length > 0 && (
              <div>
                <p className="mb-1 text-xs font-medium uppercase tracking-wide text-slate-500">
                  Children
                </p>
                <div className="flex flex-wrap gap-2">
                  {childLinks.map((link) => (
                    <RelationChip
                      key={link.relationshipId}
                      link={link}
                      canEdit={canEdit}
                      unlinking={unlinkingId === link.relationshipId}
                      onUnlink={() => void unlinkRelationship(link.relationshipId)}
                    />
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {(linkError || saveError || familyActionError) && (
          <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            {saveError || linkError || familyActionError}
          </div>
        )}

        {canEdit && (
          <div className="mb-4 rounded-xl border border-slate-200 p-3">
            <p className="mb-2 text-sm font-medium text-slate-700">Families</p>
            <p className="mb-3 text-xs text-slate-500">
              One person record can belong to multiple families. Use{" "}
              <span className="font-medium text-[#5c3d6e]">Married in</span> when they joined a
              family through marriage (not their birth family). The app may auto-label someone as
              native when their patronymic matches the family name — override that here.
            </p>
            {familiesLoading ? (
              <p className="text-sm text-slate-500">Loading families...</p>
            ) : personFamilies.length === 0 ? (
              <p className="text-sm text-slate-500">Not assigned to any family yet.</p>
            ) : (
              <div className="mb-3 space-y-2">
                {personFamilies.map((f) => (
                  <div
                    key={f.id}
                    className="flex flex-wrap items-center gap-2 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2"
                  >
                    <span className="text-sm font-medium text-slate-800">{f.name}</span>
                    <span
                      className={`rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${
                        f.married_in
                          ? "bg-[#ede4f6] text-[#5c3d6e]"
                          : "bg-white text-slate-500 ring-1 ring-slate-200"
                      }`}
                    >
                      {f.married_in ? "Married in" : "Native"}
                    </span>
                    <button
                      type="button"
                      onClick={() => void setMarriedInForFamily(f.id, !f.married_in)}
                      disabled={togglingFamilyId === f.id}
                      className="rounded-full border border-slate-200 bg-white px-2.5 py-0.5 text-xs text-slate-600 hover:border-[#8f6bab] hover:text-[#5c3d6e] disabled:opacity-50"
                    >
                      {togglingFamilyId === f.id
                        ? "…"
                        : f.married_in
                          ? "Mark as native"
                          : "Mark as married in"}
                    </button>
                    <button
                      type="button"
                      onClick={() => void removeFromFamily(f.id)}
                      disabled={removingFamilyId === f.id}
                      aria-label={`Remove from ${f.name}`}
                      title="Remove from family"
                      className="ml-auto rounded-full px-1.5 text-slate-400 hover:bg-red-50 hover:text-red-600 disabled:opacity-50"
                    >
                      {removingFamilyId === f.id ? "…" : "×"}
                    </button>
                  </div>
                ))}
              </div>
            )}
            {addableFamilies.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {addableFamilies.map((f) => (
                  <button
                    key={f.id}
                    type="button"
                    onClick={() => void addToFamily(f.id)}
                    disabled={addingFamilyId === f.id}
                    className="rounded-full border border-slate-200 bg-white px-3 py-1 text-xs text-slate-600 hover:border-accent hover:text-accent disabled:opacity-50"
                  >
                    {addingFamilyId === f.id ? "Adding..." : `+ ${f.name}`}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {!canEdit && personFamilies.length > 0 && (
          <div className="mb-4 rounded-xl border border-slate-200 bg-slate-50 p-3">
            <p className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500">
              Families
            </p>
            <p className="text-sm text-slate-700">
              {personFamilies.map((f) => f.name).join(", ")}
            </p>
          </div>
        )}

        {canEdit && (
          <div className="space-y-3">
            <input
              className="w-full rounded-lg border border-slate-200 px-3 py-2"
              placeholder="Given name"
              value={form.given_name}
              onChange={(e) => setForm({ ...form, given_name: e.target.value })}
            />
            <div className="flex gap-2">
              <input
                className="w-full rounded-lg border border-slate-200 px-3 py-2"
                placeholder="Patronymic"
                value={form.patronymic}
                onChange={(e) => setForm({ ...form, patronymic: e.target.value })}
              />
              <button
                onClick={() => void suggestPatronymic()}
                className="rounded-lg border border-slate-200 px-3 text-sm text-slate-600"
              >
                From father
              </button>
            </div>
            <input
              className="w-full rounded-lg border border-slate-200 px-3 py-2"
              placeholder="Clan name (optional)"
              value={form.clan_name}
              onChange={(e) => setForm({ ...form, clan_name: e.target.value })}
            />
            <select
              className="w-full rounded-lg border border-slate-200 px-3 py-2"
              value={form.gender}
              onChange={(e) => setForm({ ...form, gender: e.target.value })}
            >
              <option value="">Gender</option>
              <option value="male">Male</option>
              <option value="female">Female</option>
            </select>

            <div className="rounded-xl border border-slate-200 p-3">
              <label className="flex cursor-pointer items-center gap-3">
                <input
                  type="checkbox"
                  checked={Boolean(form.deceased)}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      deceased: e.target.checked,
                      ...(e.target.checked ? {} : { death_date: undefined, death_place: "" }),
                    })
                  }
                  className="h-4 w-4 rounded border-slate-300 text-accent focus:ring-accent"
                />
                <span className="text-sm font-medium text-slate-700">Deceased</span>
              </label>
              <div className="mt-3 grid gap-2 sm:grid-cols-2">
                <label className="block text-sm text-slate-600">
                  Birth date
                  <input
                    type="date"
                    className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2"
                    value={form.birth_date?.slice(0, 10) || ""}
                    onChange={(e) =>
                      setForm({ ...form, birth_date: e.target.value || undefined })
                    }
                  />
                </label>
                <label className={`block text-sm text-slate-600 ${form.deceased ? "" : "opacity-50"}`}>
                  Death date
                  <input
                    type="date"
                    disabled={!form.deceased}
                    className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 disabled:bg-slate-50"
                    value={form.death_date?.slice(0, 10) || ""}
                    onChange={(e) =>
                      setForm({ ...form, death_date: e.target.value || undefined })
                    }
                  />
                </label>
                <input
                  className="rounded-lg border border-slate-200 px-3 py-2 sm:col-span-2"
                  placeholder="Birth place"
                  value={form.birth_place || ""}
                  onChange={(e) => setForm({ ...form, birth_place: e.target.value })}
                />
                <input
                  className={`rounded-lg border border-slate-200 px-3 py-2 sm:col-span-2 ${form.deceased ? "" : "opacity-50"}`}
                  placeholder="Death place"
                  disabled={!form.deceased}
                  value={form.death_place || ""}
                  onChange={(e) => setForm({ ...form, death_place: e.target.value })}
                />
              </div>
            </div>

            <textarea
              className="w-full rounded-lg border border-slate-200 px-3 py-2"
              placeholder="Notes"
              rows={3}
              value={form.notes}
              onChange={(e) => setForm({ ...form, notes: e.target.value })}
            />
            <label className="block">
              <span className="text-sm text-slate-600">Photo</span>
              <input
                type="file"
                accept="image/*"
                className="mt-1 block w-full text-sm"
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) void uploadPhoto(file);
                }}
              />
            </label>

            <div className="rounded-xl border border-slate-200 p-3">
              <p className="mb-2 text-sm font-medium text-slate-700">Link to existing person</p>
              <div className="mb-3 flex flex-wrap gap-2">
                {(["parent", "child", "spouse"] as const).map((mode) => (
                  <button
                    key={mode}
                    onClick={() => setLinkMode(mode)}
                    className={`rounded-full px-3 py-1 text-xs capitalize ${
                      linkMode === mode
                        ? "bg-accent text-white"
                        : "bg-slate-100 text-slate-600"
                    }`}
                  >
                    As {mode}
                  </button>
                ))}
              </div>

              <div className="mb-3 flex gap-2">
                {(["family", "all"] as const).map((source) => (
                  <button
                    key={source}
                    onClick={() => setLinkSource(source)}
                    className={`rounded-full px-3 py-1 text-xs ${
                      linkSource === source
                        ? "bg-slate-800 text-white"
                        : "bg-slate-100 text-slate-600"
                    }`}
                  >
                    {source === "family" ? "This family" : "Other families"}
                  </button>
                ))}
              </div>

              {linkSource === "family" ? (
                emptyLinkMessage() ? (
                  <p className="text-sm text-slate-500">{emptyLinkMessage()}</p>
                ) : (
                  <ul className="max-h-40 space-y-1 overflow-y-auto">
                    {linkableForMode.map((p) => (
                      <li key={p.id}>
                        <button
                          type="button"
                          onClick={() => void linkRelationship(p.id, linkMode, true)}
                          className="flex w-full items-center justify-between rounded-lg border border-slate-200 bg-white px-3 py-2 text-left text-sm hover:border-accent hover:bg-slate-50"
                        >
                          <span>{displayName(p)}</span>
                          <span className="text-xs text-accent">Link</span>
                        </button>
                      </li>
                    ))}
                  </ul>
                )
              ) : (
                <div className="space-y-2">
                  <input
                    className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm"
                    placeholder="Search people in your other families..."
                    value={globalQuery}
                    onChange={(e) => setGlobalQuery(e.target.value)}
                  />
                  {globalSearching && (
                    <p className="text-sm text-slate-500">Searching...</p>
                  )}
                  {!globalSearching && globalQuery.trim() && linkableGlobalResults.length === 0 && (
                    <p className="text-sm text-slate-500">
                      {linkMode === "spouse"
                        ? "No unmarried people match that search."
                        : "No matching people found."}
                    </p>
                  )}
                  {!globalSearching && linkableGlobalResults.length > 0 && (
                    <ul className="max-h-48 space-y-1 overflow-y-auto">
                      {linkableGlobalResults.map((hit) => (
                        <li key={hit.person.id}>
                          <button
                            type="button"
                            onClick={() =>
                              void linkRelationship(hit.person.id, linkMode, hit.in_target_family)
                            }
                            className="flex w-full items-center justify-between gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-left text-sm hover:border-accent hover:bg-slate-50"
                          >
                            <div className="min-w-0">
                              <div className="truncate">{displayName(hit.person)}</div>
                              <div className="truncate text-xs text-slate-500">
                                {hit.families
                                  .filter((f) => f.id !== familyId)
                                  .map((f) => f.name)
                                  .join(", ") || "Other family"}
                              </div>
                            </div>
                            <span className="shrink-0 text-xs text-accent">
                              {hit.in_target_family ? "Link" : "Add & link"}
                            </span>
                          </button>
                        </li>
                      ))}
                    </ul>
                  )}
                  {!globalQuery.trim() && (
                    <p className="text-sm text-slate-500">
                      Search across all families you have access to. Linking reuses the
                      existing person and adds them to this family. Spouses appear in both
                      family trees.
                    </p>
                  )}
                </div>
              )}

              {linkMode === "parent" && (
                <>
                  <button
                    onClick={() => setShowNewParent((v) => !v)}
                    className="mt-3 text-sm text-accent hover:underline"
                  >
                    {showNewParent ? "Cancel new parent" : "+ Create new parent"}
                  </button>
                  {showNewParent && (
                    <div className="mt-2 grid gap-2 sm:grid-cols-3">
                      <input
                        className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
                        placeholder="Given name"
                        value={newParent.given_name}
                        onChange={(e) =>
                          setNewParent({ ...newParent, given_name: e.target.value })
                        }
                      />
                      <input
                        className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
                        placeholder="Patronymic"
                        value={newParent.patronymic}
                        onChange={(e) =>
                          setNewParent({ ...newParent, patronymic: e.target.value })
                        }
                      />
                      <select
                        className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
                        value={newParent.gender}
                        onChange={(e) =>
                          setNewParent({ ...newParent, gender: e.target.value })
                        }
                      >
                        <option value="male">Male</option>
                        <option value="female">Female</option>
                      </select>
                      <button
                        onClick={() => void createAndLinkParent()}
                        className="rounded-lg bg-accent px-3 py-2 text-sm text-white sm:col-span-3"
                      >
                        Create &amp; link parent
                      </button>
                    </div>
                  )}
                </>
              )}
            </div>

            <button
              disabled={saving}
              onClick={() => void save()}
              className="w-full rounded-lg bg-accent py-2.5 font-medium text-white hover:bg-accent-hover disabled:opacity-50"
            >
              {saving ? "Saving..." : "Save"}
            </button>
          </div>
        )}

        {!canEdit && (
          <div className="space-y-2 text-sm text-slate-600">
            <p>Patronymic: {currentPersonRecord.patronymic || "—"}</p>
            <p>Clan: {currentPersonRecord.clan_name || "—"}</p>
            {isDeceased(currentPersonRecord) && (
              <p className="font-medium text-slate-500">
                Deceased{lifeSpan(currentPersonRecord) ? ` · ${lifeSpan(currentPersonRecord)}` : ""}
              </p>
            )}
            {currentPersonRecord.birth_place && <p>Born: {currentPersonRecord.birth_place}</p>}
            {currentPersonRecord.death_place && <p>Died: {currentPersonRecord.death_place}</p>}
            <p>Notes: {currentPersonRecord.notes || "—"}</p>
          </div>
        )}
      </div>
    </div>
  );
}

function RelationChip({
  link,
  canEdit,
  unlinking,
  divorcing,
  onDivorce,
  onUnlink,
}: {
  link: LinkedRelation;
  canEdit: boolean;
  unlinking: boolean;
  divorcing?: boolean;
  onDivorce?: () => void;
  onUnlink: () => void;
}) {
  const name = relationLabel(link);
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full py-1 pl-3 pr-1 text-sm shadow-sm ${
        link.divorced ? "bg-slate-100 text-slate-500" : "bg-white text-slate-700"
      }`}
    >
      <span>{name}{link.divorced ? " (divorced)" : ""}</span>
      {canEdit && onDivorce && !link.divorced && (
        <button
          type="button"
          onClick={onDivorce}
          disabled={divorcing || unlinking}
          title="Mark as divorced"
          className="rounded-full px-1.5 text-xs text-slate-500 hover:bg-amber-50 hover:text-amber-700 disabled:opacity-50"
        >
          {divorcing ? "…" : "divorce"}
        </button>
      )}
      {canEdit && (
        <button
          type="button"
          onClick={onUnlink}
          disabled={unlinking || divorcing}
          aria-label={`Unlink ${name}`}
          title="Unlink"
          className="rounded-full px-1.5 text-slate-400 hover:bg-red-50 hover:text-red-600 disabled:opacity-50"
        >
          {unlinking ? "…" : "×"}
        </button>
      )}
    </span>
  );
}