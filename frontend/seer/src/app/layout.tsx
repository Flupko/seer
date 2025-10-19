import Drawer from "@/ui/drawer/Drawer";
import { ModalContainer } from "@/ui/modal/Modal";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import localFont from "next/font/local";
import { Suspense } from "react";
import { ToastContainer } from "react-toastify";
import Navbar from "../ui/navbar/Navbar";
import "./globals.css";
import ReactQueryProvider from "./ReactQueryProvider";
import URLHandler from "./URLHandler";
import { WsProvider } from "./WsProvider";

export const aeonik = localFont({
  src: [
    // Thin
    { path: "./fonts/Aeonik-Thin.woff2", weight: "100", style: "normal" },
    { path: "./fonts/Aeonik-ThinItalic.woff2", weight: "100", style: "italic" },

    // Air (treat as ExtraLight)
    { path: "./fonts/Aeonik-Air.woff2", weight: "200", style: "normal" },
    { path: "./fonts/Aeonik-AirItalic.woff2", weight: "200", style: "italic" },

    // Light
    { path: "./fonts/Aeonik-Light.woff2", weight: "300", style: "normal" },
    { path: "./fonts/Aeonik-LightItalic.woff2", weight: "300", style: "italic" },

    // Regular
    { path: "./fonts/Aeonik-Regular.woff2", weight: "400", style: "normal" },
    { path: "./fonts/Aeonik-RegularItalic.woff2", weight: "400", style: "italic" },

    // Medium
    { path: "./fonts/Aeonik-Medium.woff2", weight: "500", style: "normal" },
    { path: "./fonts/Aeonik-MediumItalic.woff2", weight: "500", style: "italic" },

    // Bold
    { path: "./fonts/Aeonik-Bold.woff2", weight: "700", style: "normal" },
    { path: "./fonts/Aeonik-BoldItalic.woff2", weight: "700", style: "italic" },

    // Black
    { path: "./fonts/Aeonik-Black.woff2", weight: "900", style: "normal" },
    { path: "./fonts/Aeonik-BlackItalic.woff2", weight: "900", style: "italic" },
  ],
  display: "swap",
  variable: "--font-aeonik",
  preload: true,
});

// const roboto = Geist({
//   subsets: ["latin"],
//   variable: "--font-sans",
//   display: "swap",
//   weight: ["400", "500", "600", "700", '800'],
// });

// GEIST, MANROPE, ROBOTTO


// export const hostGrotesk = Host_Grotesk({
//   subsets: ['latin'],
//   weight: ['400', '500', '600', '700', '800'],
//   variable: '--font-host-grotesk',
// })



export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className={`${aeonik.className} antialiased`}>
        <ReactQueryProvider>
          <ReactQueryDevtools initialIsOpen={false} />
          <WsProvider>
            <Suspense fallback={null}>
              <URLHandler />
            </Suspense>

            <ModalContainer />
            <ToastContainer aria-label={"top-left"} position="top-left" theme="dark" limit={1} />

            <div className="flex h-screen overflow-hidden relative">

              <div className="flex-1 flex flex-col min-w-0">
                <Navbar />
                <main className="flex-1 overflow-auto">
                  {children}
                </main>
              </div>

              <Drawer />

            </div>
          </WsProvider>
        </ReactQueryProvider>
      </body>
    </html>
  );
}