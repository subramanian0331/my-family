import type { Relationship, RelationshipMetadata } from "../types";

export type MaritalStatus = "married" | "divorced";

function parseMetadata(metadata: Relationship["metadata"]): RelationshipMetadata {
  if (!metadata) return {};
  if (typeof metadata === "string") {
    try {
      return JSON.parse(metadata) as RelationshipMetadata;
    } catch {
      return {};
    }
  }
  return metadata;
}

export function maritalStatus(rel: Relationship): MaritalStatus {
  return parseMetadata(rel.metadata).marital_status === "divorced" ? "divorced" : "married";
}

/** Relationships involving personId; optionally require the other person to be in memberIds. */
export function relationshipsForPerson(
  personId: string,
  relationships: Relationship[],
  memberIds?: Set<string>,
): Relationship[] {
  return relationships.filter((r) => {
    if (r.from_person_id !== personId && r.to_person_id !== personId) return false;
    if (!memberIds) return true;
    const otherId = r.from_person_id === personId ? r.to_person_id : r.from_person_id;
    return memberIds.has(otherId);
  });
}

export function isActiveSpouseRelationship(rel: Relationship): boolean {
  return rel.type === "spouse" && maritalStatus(rel) !== "divorced";
}

export function isDivorcedSpouseRelationship(rel: Relationship): boolean {
  return rel.type === "spouse" && maritalStatus(rel) === "divorced";
}