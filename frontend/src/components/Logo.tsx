type LogoProps = {
  className?: string;
  variant?: "header" | "hero" | "home";
};

export function Logo({ className = "", variant = "header" }: LogoProps) {
  const sizeClass =
    variant === "home"
      ? "h-auto w-full max-w-3xl sm:max-w-4xl lg:max-w-5xl"
      : variant === "hero"
        ? "h-auto w-full max-w-[min(100%,22rem)] sm:max-w-[26rem]"
        : "h-14 w-auto sm:h-[4.25rem]";

  return (
    <img
      src="/logo.png"
      alt="My Family — Discover Your Ancestry | Connect with Kin"
      className={`${sizeClass} ${className}`.trim()}
      decoding="async"
    />
  );
}