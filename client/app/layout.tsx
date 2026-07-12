import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "GeoGuess",
  description: "A geography guessing game.",
};

type RootLayoutProps = Readonly<{
  children: React.ReactNode;
}>;

export default function RootLayout({ children }: RootLayoutProps) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
