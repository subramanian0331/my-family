type BrandMarkProps = {
  className?: string;
};

export function BrandMark({ className = "" }: BrandMarkProps) {
  return (
    <span className={`inline-flex items-center gap-3 ${className}`.trim()}>
      <img
        src="/logo-tree.png"
        alt=""
        className="h-[2.1rem] w-auto shrink-0 object-contain sm:h-[2.5rem]"
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