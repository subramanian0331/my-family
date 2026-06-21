import type { Person } from "../types";
import { isDeceased, lifeSpan } from "../lib/personUtils";
import { PersonPhoto } from "./PersonPhoto";

export function displayName(person: Person) {
  const parts = [person.given_name, person.patronymic].filter(Boolean);
  return parts.join(" ") || "Unknown";
}

export function PersonCard({
  person,
  onClick,
  compact,
}: {
  person: Person;
  onClick?: () => void;
  compact?: boolean;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-3 rounded-xl border border-slate-200 bg-white text-left shadow-sm transition hover:border-accent/40 hover:shadow ${
        compact ? "p-2" : "p-3"
      }`}
    >
      {person.photo_id ? (
        <PersonPhoto
          photoId={person.photo_id}
          alt=""
          className={`rounded-full object-cover ${compact ? "h-8 w-8" : "h-12 w-12"}`}
          fallback={
            <div
              className={`flex items-center justify-center rounded-full bg-slate-100 font-medium text-slate-600 ${
                compact ? "h-8 w-8 text-xs" : "h-12 w-12"
              }`}
            >
              {person.given_name?.[0] || "?"}
            </div>
          }
        />
      ) : (
        <div
          className={`flex items-center justify-center rounded-full bg-slate-100 font-medium text-slate-600 ${
            compact ? "h-8 w-8 text-xs" : "h-12 w-12"
          }`}
        >
          {person.given_name?.[0] || "?"}
        </div>
      )}
      <div>
        <div
          className={`font-medium ${isDeceased(person) ? "text-slate-600" : "text-slate-900"} ${compact ? "text-sm" : ""}`}
        >
          {displayName(person)}
          {isDeceased(person) && (
            <span className="ml-1 text-xs font-normal text-slate-400">†</span>
          )}
        </div>
        {lifeSpan(person) && <div className="text-xs text-slate-500">{lifeSpan(person)}</div>}
        {person.clan_name && <div className="text-xs text-slate-500">{person.clan_name}</div>}
      </div>
    </button>
  );
}