import Image from "next/image";

export type LandingFooterLink = {
  label: string;
  href: string;
};

export type LandingFooterColumn = {
  title: string;
  links: LandingFooterLink[];
};

type LandingFooterProps = {
  brandName: string;
  tagline: string;
  columns: LandingFooterColumn[];
  copyright: string;
  logoSrc: string;
  homeHref: string;
  navigationLabel: string;
};

export function LandingFooter({
  brandName,
  tagline,
  columns,
  copyright,
  logoSrc,
  homeHref,
  navigationLabel,
}: LandingFooterProps) {
  return (
    <footer className="relative w-full overflow-hidden bg-zinc-950 font-sans text-zinc-100 antialiased">
      <div
        className="absolute inset-0 bg-[radial-gradient(circle_at_50%_0%,rgba(37,99,235,0.28),transparent_56%),linear-gradient(180deg,#111827_0%,#090a0e_72%)]"
        aria-hidden="true"
      />
      <div className="relative mx-auto flex min-h-[420px] flex-col justify-end px-4 pt-16 pb-7 sm:min-h-[500px] sm:px-12 sm:pt-20 sm:pb-8 lg:min-h-[560px] lg:pt-28">
        <div className="pointer-events-none absolute top-[12%] left-1/2 -translate-x-1/2 text-[clamp(4rem,17vw,14rem)] leading-none font-semibold tracking-[-0.08em] whitespace-nowrap text-white/[0.04] uppercase">
          {brandName}
        </div>

        <div className="relative z-10 border-t border-white/10 pt-9 sm:pt-11">
          <div className="grid gap-10 lg:grid-cols-[minmax(220px,1fr)_minmax(520px,0.98fr)] lg:gap-x-20">
            <div className="max-w-2xl">
              <a
                href={homeHref}
                className="group inline-flex min-h-10 items-start gap-2 text-zinc-50 transition-opacity duration-200 hover:opacity-85 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-indigo-400"
                aria-label={`${brandName} home`}
              >
                <Image
                  src={logoSrc}
                  alt=""
                  width={32}
                  height={32}
                  className="size-8 -translate-y-2 object-contain"
                />
                <span className="text-xl leading-none font-normal tracking-wide">
                  {brandName}
                </span>
              </a>
              <p className="max-w-lg text-sm leading-relaxed font-normal text-pretty whitespace-pre-line text-zinc-300/80">
                {tagline}
              </p>
            </div>

            <nav
              aria-label={navigationLabel}
              className="grid grid-cols-1 gap-7 min-[520px]:grid-cols-3 min-[520px]:gap-x-10 lg:gap-x-[66px]"
            >
              {columns.map((column) => (
                <div key={column.title}>
                  <h2 className="text-sm leading-none font-light tracking-wide text-zinc-50 uppercase">
                    {column.title}
                  </h2>
                  <ul className="mt-4 space-y-2">
                    {column.links.map((link) => (
                      <li key={link.label}>
                        <a
                          href={link.href}
                          className="inline-flex min-h-5 items-center text-sm leading-tight font-light text-zinc-300/75 transition-colors duration-200 hover:text-zinc-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-400"
                        >
                          {link.label}
                        </a>
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </nav>
          </div>

          <p className="mt-9 border-t border-white/10 pt-4 text-sm font-normal text-zinc-400/80">
            {copyright}
          </p>
        </div>
      </div>
    </footer>
  );
}
