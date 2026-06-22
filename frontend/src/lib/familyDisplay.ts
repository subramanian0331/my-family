import type { FamilySummary } from "../types";

const PLACEHOLDER_DESCRIPTIONS = new Set([
  "",
  "no description",
  "auto-created from family name",
]);

export function familyInitials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return `${parts[0][0] ?? ""}${parts[parts.length - 1][0] ?? ""}`.toUpperCase();
}

export function familySubtitle(family: FamilySummary): string | null {
  const description = family.description?.trim();
  if (!description || PLACEHOLDER_DESCRIPTIONS.has(description.toLowerCase())) {
    return null;
  }
  return description;
}

export function formatRelativeTime(iso?: string): string | null {
  if (!iso) return null;
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return null;

  const diffMs = Date.now() - date.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  if (diffDays <= 0) return "Updated today";
  if (diffDays === 1) return "Updated yesterday";
  if (diffDays < 7) return `Updated ${diffDays} days ago`;
  if (diffDays < 30) return `Updated ${Math.floor(diffDays / 7)} weeks ago`;
  return `Updated ${date.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" })}`;
}

export function memberCountLabel(count?: number): string {
  const n = count ?? 0;
  return n === 1 ? "1 member" : `${n} members`;
}