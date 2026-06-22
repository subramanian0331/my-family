export function HomeBackground() {
  return (
    <div className="pointer-events-none fixed inset-x-0 top-14 z-0 h-[min(34vh,20rem)] overflow-hidden" aria-hidden>
      <div className="absolute inset-0 bg-gradient-to-b from-[#eef7f3] via-brand-cream to-brand-cream" />
      <img
        src="/logo-tree.png"
        alt=""
        className="absolute left-1/2 top-[52%] h-[min(72%,18rem)] w-auto max-w-[min(92vw,36rem)] -translate-x-1/2 -translate-y-1/2 object-contain opacity-95"
        decoding="async"
      />
      <div className="absolute inset-0 bg-gradient-to-b from-white/5 via-transparent to-brand-cream" />
      <div className="absolute inset-x-0 bottom-0 h-24 bg-gradient-to-b from-transparent to-brand-cream" />
    </div>
  );
}