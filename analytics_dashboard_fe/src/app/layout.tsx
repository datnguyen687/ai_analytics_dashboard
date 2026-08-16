import type { Metadata } from "next";
import "./globals.css";
import { AppFrame } from "@/components/app-frame";
import { AuthProvider } from "@/lib/auth-context";

export const metadata: Metadata = {
  title: "Logistics Intelligence",
  description: "AI-powered logistics analytics dashboard",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body className="antialiased">
        <AuthProvider>
          <AppFrame>{children}</AppFrame>
        </AuthProvider>
      </body>
    </html>
  );
}
