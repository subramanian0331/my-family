export function HomeBackground() {
  return (
    <div className="pointer-events-none fixed inset-0 z-0 overflow-hidden" aria-hidden>
      <div className="absolute inset-0 bg-gradient-to-br from-brand-cream via-brand-mist to-[#d8eaf4]" />
      <div
        className="absolute inset-x-0 top-[-4%] h-[min(88vh,52rem)] bg-[url('/logo-hero.png')] bg-contain bg-top bg-no-repeat opacity-[0.42] mix-blend-multiply sm:opacity-[0.48]"
        style={{ backgroundSize: "min(92vw, 56rem) auto" }}
      />
      <div className="absolute inset-0 bg-gradient-to-b from-white/10 via-brand-cream/25 to-brand-cream/75" />
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center_top,rgba(255,255,255,0.15),transparent_55%)]" />
    </div>
  );
}