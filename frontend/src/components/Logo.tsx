type LogoProps = {
  className?: string;
  variant?: "header" | "hero";
};

export function Logo({ className = "", variant = "header" }: LogoProps) {
  const sizeClass =
    variant === "hero"
      ? "h-auto w-full max-w-[280px]"
      : "h-10 w-auto sm:h-11";

  return (
    <img
      src="/logo.png"
      alt="My Family — Discover Your Ancestry | Connect with Kin"
      className={`${sizeClass} ${className}`.trim()}
      decoding="async"
    />
  );
}