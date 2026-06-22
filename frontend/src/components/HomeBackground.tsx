export function HomeBackground() {
  return (
    <div
      className="pointer-events-none fixed inset-x-0 top-14 z-0 h-[min(46vh,28rem)] sm:h-[min(50vh,32rem)]"
      aria-hidden
    >
      <div className="absolute inset-0 bg-gradient-to-b from-[#eef7f3] via-brand-cream to-brand-cream" />
      <img
        src="/logo.png"
        alt=""
        className="absolute left-1/2 top-0 h-full w-auto max-w-[min(96vw,44rem)] -translate-x-1/2 object-contain object-top"
        decoding="async"
        fetchPriority="high"
      />
      <div className="absolute inset-x-0 bottom-0 h-12 bg-gradient-to-b from-transparent via-brand-cream/80 to-brand-cream sm:h-14" />
    </div>
  );
}