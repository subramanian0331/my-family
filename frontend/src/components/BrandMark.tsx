type BrandMarkProps = {
  className?: string;
};

export function BrandMark({ className = "" }: BrandMarkProps) {
  return (
    <span className={`inline-flex items-center gap-2.5 ${className}`.trim()}>
      <span className="relative h-9 w-9 shrink-0 overflow-hidden rounded-lg sm:h-10 sm:w-10">
        <img
          src="/logo.png"
          alt=""
          className="absolute left-1/2 top-0 h-[3.75rem] w-auto max-w-none -translate-x-1/2 object-cover object-top sm:h-[4.25rem]"
          decoding="async"
          aria-hidden
        />
      </span>
      <span className="font-brand text-lg leading-none tracking-tight sm:text-xl">
        <span className="font-medium text-brand-my">My </span>
        <span className="font-bold text-brand-family">Family</span>
      </span>
    </span>
  );
}