import type { Person, Relationship } from "../types";
import { isActiveSpouseRelationship } from "./relationshipMetadata";

/** Person IDs that already appear in an active spouse relationship. */
export function takenSpouseIds(relationships: Relationship[]): Set<string> {
  const ids = new Set<string>();
  for (const rel of relationships) {
    if (!isActiveSpouseRelationship(rel)) continue;
    ids.add(rel.from_person_id);
    ids.add(rel.to_person_id);
  }
  return ids;
}

export function personHasSpouse(
  personId: string,
  relationships: Relationship[],
  person?: Pick<Person, "has_spouse">,
): boolean {
  if (person?.has_spouse) return true;
  return takenSpouseIds(relationships).has(personId);
}

/** Whether this person can be newly linked as a spouse of `forPersonId`. */
export function isAvailableSpousePartner(
  candidateId: string,
  forPersonId: string,
  relationships: Relationship[],
  candidate?: Pick<Person, "has_spouse">,
): boolean {
  if (candidateId === forPersonId) return false;
  if (personHasSpouse(forPersonId, relationships)) return false;
  if (personHasSpouse(candidateId, relationships, candidate)) return false;
  return true;
}

/** Whether these two people already have any spouse link (active or divorced). */
export function alreadyLinkedAsSpouses(
  aId: string,
  bId: string,
  relationships: Relationship[],
): boolean {
  for (const rel of relationships) {
    if (rel.type !== "spouse") continue;
    const pair = [rel.from_person_id, rel.to_person_id].sort().join(":");
    const target = [aId, bId].sort().join(":");
    if (pair === target) return true;
  }
  return false;
}