import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "GeoGuessr",
  description: "Guess the location shown on the map. A minimal geography guessing game.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      {/* Immersive GeoGuessr layout: fixed, non-scrolling dark canvas so the
          Street View panorama can fill the viewport behind the overlays. */}
      <body className="h-screen w-screen overflow-hidden bg-zinc-900 text-white">
        {children}
      </body>
    </html>
  );
}
