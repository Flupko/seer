import "./globals.css";
import { Host_Grotesk, Inter, Roboto } from "next/font/google";
import ReactQueryProvider from "./ReactQueryProvider";
import Navbar from "./Navbar";
import { ModalContainer, ModalProvider } from "@/ui/modal/Modal";
import AuthProvider from "./AuthProvider";
import { ToastContainer } from "react-toastify";
import URLHandler from "./URLHandler";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";

const roboto = Host_Grotesk({
  subsets: ["latin"],
  variable: "--font-sans",
  display: "swap",
  weight: ["300", "400", "500", "600", "700", "800"],
});


export const hostGrotesk = Host_Grotesk({
  subsets: ['latin'],
  weight: ['400', '500', '600', '700', '800'],
  variable: '--font-host-grotesk',
})

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${roboto.className} ${hostGrotesk.variable} antialiased `}
      >
        <ReactQueryProvider>
          <ReactQueryDevtools initialIsOpen={true} />
          <ModalProvider>
            <URLHandler />
            <ModalContainer />
            <ToastContainer aria-label={"top-left"} position="top-left" theme="dark" limit={1} />
            <Navbar />
            {children}
          </ModalProvider>
        </ReactQueryProvider>
      </body>
    </html>
  );
}
