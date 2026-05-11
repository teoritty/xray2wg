export function Topbar({ title }: { title: string }) {
  return (
    <header className="sticky top-0 z-20 border-b border-[#2a2a3f] bg-[#0d0d12]/95 px-8 py-4 backdrop-blur">
      <h1 className="text-lg font-semibold text-[#e2e8f0]">{title}</h1>
    </header>
  );
}
