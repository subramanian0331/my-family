type BrandMarkProps = {
  className?: string;
};

export function BrandMark({ className = "" }: BrandMarkProps) {
  return (
    <span className={`inline-flex items-center gap-2.5 ${className}`.trim()}>
      <img
        src="/logo-tree.png"
        alt=""
        className="h-9 w-9 shrink-0 object-contain object-center sm:h-10 sm:w-10"
        decoding="async"
        aria-hidden
      />
      <span className="font-brand text-lg leading-none tracking-tight sm:text-xl">
        <span className="font-medium text-brand-my">My </span>
        <span className="font-bold text-brand-family">Family</span>
      </span>
    </span>
  );
}