type LogoProps = {
  className?: string;
  variant?: "header" | "hero" | "home";
};

export function Logo({ className = "", variant = "header" }: LogoProps) {
  const isHome = variant === "home";
  const sizeClass = isHome
    ? "h-auto w-full max-w-[17rem] sm:max-w-[19rem] lg:max-w-[21rem]"
    : variant === "hero"
      ? "h-auto w-full max-w-[min(100%,22rem)] sm:max-w-[26rem]"
      : "h-14 w-auto sm:h-[4.25rem]";

  return (
    <img
      src={isHome ? "/logo-hero.png" : "/logo.png"}
      alt="My Family — Discover Your Ancestry | Connect with Kin"
      className={`${sizeClass} ${className}`.trim()}
      decoding="async"
      fetchPriority={isHome ? "high" : undefined}
    />
  );
}