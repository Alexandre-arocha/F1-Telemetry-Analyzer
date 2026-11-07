/* eslint-disable @next/next/no-img-element -- remote headshots from the data provider, not worth the image optimizer */
export function DriverAvatar({
  src,
  name,
  colour,
  size = 40,
}: {
  src?: string;
  name: string;
  colour?: string;
  size?: number;
}) {
  const ring = colour ? `#${colour}` : "#555";
  const initials = name
    .split(" ")
    .map((p) => p[0])
    .join("")
    .slice(0, 2);
  return (
    <span
      className="inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted text-xs font-medium"
      style={{ width: size, height: size, boxShadow: `0 0 0 2px ${ring}` }}
    >
      {src ? <img src={src} alt={name} loading="lazy" className="h-full w-full object-cover object-top" /> : initials}
    </span>
  );
}
