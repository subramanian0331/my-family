import { useEffect, useState } from "react";
import { Logo } from "./Logo";

const FADE_DISTANCE = 280;

export function HomeHero() {
  const [scrollY, setScrollY] = useState(0);

  useEffect(() => {
    const onScroll = () => setScrollY(window.scrollY);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const progress = Math.min(1, scrollY / FADE_DISTANCE);
  const fade = 1 - progress;
  const lift = scrollY * 0.35;
  const scale = 1 - progress * 0.06;

  return (
    <div
      className="pointer-events-none sticky top-14 z-0 flex min-h-[min(30vh,15rem)] flex-col items-center justify-center px-4 pb-8 pt-5 sm:min-h-[min(34vh,17rem)] sm:pt-6"
      style={{
        opacity: fade,
        transform: `translateY(-${lift}px) scale(${scale})`,
        willChange: "opacity, transform",
      }}
      aria-hidden={fade < 0.05}
    >
      <Logo variant="home" className="mx-auto drop-shadow-sm" />
      <p
        className="mt-3 max-w-sm text-center text-sm text-brand-blue/85 sm:mt-4 sm:text-base"
        style={{ opacity: fade }}
      >
        Select a family to view the tree and search people.
      </p>
    </div>
  );
}