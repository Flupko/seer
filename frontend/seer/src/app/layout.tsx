import Drawer from "@/ui/drawer/Drawer";
import { ModalContainer } from "@/ui/modal/Modal";
import NavbarMobile from "@/ui/navbar/NavbarMobile";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { Viewport } from "next";
import { Inter } from "next/font/google";
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
    { path: "./fonts/aeonik/Aeonik-Thin.woff2", weight: "100", style: "normal" },
    { path: "./fonts/aeonik/Aeonik-ThinItalic.woff2", weight: "100", style: "italic" },

    // Air (treat as ExtraLight)
    { path: "./fonts/aeonik/Aeonik-Air.woff2", weight: "200", style: "normal" },
    { path: "./fonts/aeonik/Aeonik-AirItalic.woff2", weight: "200", style: "italic" },

    // Light
    { path: "./fonts/aeonik/Aeonik-Light.woff2", weight: "300", style: "normal" },
    { path: "./fonts/aeonik/Aeonik-LightItalic.woff2", weight: "300", style: "italic" },
    // Regular
    { path: "./fonts/aeonik/Aeonik-Regular.woff2", weight: "400", style: "normal" },
    { path: "./fonts/aeonik/Aeonik-RegularItalic.woff2", weight: "400", style: "italic" },

    // Medium
    { path: "./fonts/aeonik/Aeonik-Medium.woff2", weight: "500", style: "normal" },
    { path: "./fonts/aeonik/Aeonik-MediumItalic.woff2", weight: "500", style: "italic" },
    // Bold
    { path: "./fonts/aeonik/Aeonik-Bold.woff2", weight: "700", style: "normal" },
    { path: "./fonts/aeonik/Aeonik-BoldItalic.woff2", weight: "700", style: "italic" },

    // Black
    { path: "./fonts/aeonik/Aeonik-Black.woff2", weight: "900", style: "normal" },
    { path: "./fonts/aeonik/Aeonik-BlackItalic.woff2", weight: "900", style: "italic" },
  ],
  display: "swap",
  variable: "--font-aeonik",
  preload: true,
});

// export const openSauceOne = localFont({
//   src: [
//     // Thin
//     { path: "./fonts/open-sauce-one/OpenSauceOne-Thin.woff2", weight: "100", style: "normal" },
//     { path: "./fonts/open-sauce-one/OpenSauceOne-ThinItalic.woff2", weight: "200", style: "italic" },

//     // Light
//     { path: "./fonts/open-sauce-one/OpenSauceOne-Light.woff2", weight: "300", style: "normal" },
//     { path: "./fonts/open-sauce-one/OpenSauceOne-LightItalic.woff2", weight: "300", style: "italic" },

//     // Regular
//     { path: "./fonts/open-sauce-one/OpenSauceOne-Regular.woff2", weight: "400", style: "normal" },
//     { path: "./fonts/open-sauce-one/OpenSauceOne-RegularItalic.woff2", weight: "400", style: "italic" },

//     // Medium
//     { path: "./fonts/open-sauce-one/OpenSauceOne-Medium.woff2", weight: "500", style: "normal" },
//     { path: "./fonts/open-sauce-one/OpenSauceOne-MediumItalic.woff2", weight: "500", style: "italic" },

//     // Bold
//     { path: "./fonts/open-sauce-one/OpenSauceOne-Bold.woff2", weight: "700", style: "normal" },
//     { path: "./fonts/open-sauce-one/OpenSauceOne-BoldItalic.woff2", weight: "700", style: "italic" },
//     // Black
//     { path: "./fonts/open-sauce-one/OpenSauceOne-Black.woff2", weight: "900", style: "normal" },
//     { path: "./fonts/open-sauce-one/OpenSauceOne-BlackItalic.woff2", weight: "900", style: "italic" },

//   ],
//   variable: "--font-open-sauce-one",
// });

// const roboto = Geist({
//   subsets: ["latin"],
//   variable: "--font-sans",
//   display: "swap",
//   weight: ["400", "500", "600", "700", '800'],
// });

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-sans',
  display: 'swap',
  weight: ["400", "500", "600", "700", '800'],
})



// GEIST, MANROPE, ROBOTTO


// export const hostGrotesk = Host_Grotesk({
//   subsets: ['latin'],
//   weight: ['400', '500', '600', '700', '800'],
//   variable: '--font-host-grotesk',
// })



export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
};



export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">

      <head>
        <meta name="theme-color" content="#080808"></meta>
      </head>

      <body className={`${inter.className} antialiased`}>
        <ReactQueryProvider>
          <ReactQueryDevtools initialIsOpen={false} />
          <WsProvider>
            <Suspense fallback={null}>
              <URLHandler />
            </Suspense>

            <ModalContainer />
            <ToastContainer aria-label={"top-left"} position="top-left" theme="dark" limit={1} />

            {/* Root layout */}
            <div className="flex w-full relative lg:overflow-hidden">
              <main className="flex flex-col relative w-full flex-1 lg:overflow-hidden">
                <Navbar />

                {/* Page content */}
                <div className="overflow-hidden lg:overflow-auto md:h-[calc(100vh-76px)] bg-grayscale-black min-h-[60vh] scrollbar-light">
                  <div className="min-h-[100vh] pt-5 md:pt-6">
                    {children}
                  </div>

                </div>

                <NavbarMobile />
              </main>

              <Drawer />
            </div>



          </WsProvider>
        </ReactQueryProvider>
      </body>
    </html>
  );
}