import type { Metadata } from "next";
import { Space_Grotesk } from "next/font/google";
import { Nav } from "@/components/nav";
import "./globals.css";
const spaceGrotesk = Space_Grotesk({
  subsets: ["latin"],
  variable: "--font-space-grotesk",
});
export const metadata: Metadata = {
  title: "DocSnap",
  description:
    "Verifiable claim intelligence — investigate claims, see the evidence, and create cryptographically verifiable proofs of what was published online.",
};
export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={`${spaceGrotesk.variable} dark`}>
      <body className="min-h-screen bg-background font-sans antialiased">
        <Nav />
        {children}
      </body>
    </html>
  );
}
