import type { Person } from "../types";

export function isDeceased(person: Pick<Person, "deceased" | "death_date">): boolean {
  return Boolean(person.deceased || person.death_date);
}

export function lifeSpan(person: Pick<Person, "birth_date" | "death_date" | "deceased">): string {
  const birth = person.birth_date?.slice(0, 4) || "";
  const death = person.death_date?.slice(0, 4) || "";
  if (birth && death) return `${birth}–${death}`;
  if (birth && isDeceased(person)) return `${birth}–`;
  if (birth) return `${birth}–`;
  if (death) return `–${death}`;
  if (isDeceased(person)) return "Deceased";
  return "";
}