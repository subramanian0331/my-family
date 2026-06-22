type CreateFamilyFormProps = {
  name: string;
  description: string;
  onNameChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onSubmit: () => void;
  compact?: boolean;
};

export function CreateFamilyForm({
  name,
  description,
  onNameChange,
  onDescriptionChange,
  onSubmit,
  compact = false,
}: CreateFamilyFormProps) {
  return (
    <div className={compact ? "space-y-3" : "space-y-4"}>
      <div className={`grid gap-3 ${compact ? "sm:grid-cols-[1fr_1.2fr_auto]" : "sm:grid-cols-2"}`}>
        <input
          className="rounded-lg border border-brand-leaf bg-white px-3 py-2.5 outline-none focus:border-brand-teal focus:ring-2 focus:ring-brand-teal/20"
          placeholder="Family name"
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
        />
        <input
          className="rounded-lg border border-brand-leaf bg-white px-3 py-2.5 outline-none focus:border-brand-teal focus:ring-2 focus:ring-brand-teal/20"
          placeholder="Description (optional)"
          value={description}
          onChange={(e) => onDescriptionChange(e.target.value)}
        />
        {compact && (
          <button
            onClick={onSubmit}
            className="rounded-lg bg-accent px-4 py-2.5 font-medium text-white shadow-sm hover:bg-accent-hover sm:px-5"
          >
            Create
          </button>
        )}
      </div>
      {!compact && (
        <button
          onClick={onSubmit}
          className="rounded-lg bg-accent px-4 py-2.5 font-medium text-white shadow-sm hover:bg-accent-hover"
        >
          Create family
        </button>
      )}
    </div>
  );
}