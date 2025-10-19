"use client"

import { useUserQuery } from "@/lib/queries/useUserQuery";
import NavbarLogged from "@/ui/navbar/NavbarLogged";
import NavbarUnlogged from "@/ui/navbar/NavbarUnlogged";
import Link from "next/link";

export default function Navbar() {
    const { data: user, isPending } = useUserQuery();

    return (
        <nav className="border-b border-gray-700 px-4 md:px-12 bg-grayscale-black z-50 sticky top-0 h-19">
            <div className="grid grid-cols-3 items-center max-w-7xl mx-auto h-full">

                <Link href="/" className="select-none cursor-pointer">
                    <h1 className="text-3xl font-bold text-white justify-self-start">SEER</h1>
                </Link>


                {!isPending && user &&
                    <NavbarLogged />
                }

                {!isPending && !user && (
                    <>
                        <div className="justify-self-center"></div>
                        <div className="justify-self-end"> <NavbarUnlogged /></div>

                    </>
                )}
            </div>
        </nav>
    );
}
