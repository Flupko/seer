"use client"

import { useUserQuery } from "@/lib/queries/useUserQuery";
import NavbarLogged from "@/ui/navbar/NavbarLogged";
import NavbarUnlogged from "@/ui/navbar/NavbarUnlogged";
import Link from "next/link";
import SearchBar from "../SearchBar";

export default function Navbar() {
    const { data: user, isPending } = useUserQuery();

    return (
        <>
            <nav className="border-b border-gray-700 w-full px-4 md:px-12 bg-grayscale-black z-50 sticky top-0 h-16">

                {/* Chrome IOS scroll hider */}
                <div className="bg-gray-900 h-12.5 w-full fixed -top-[50px]"></div>
                <div className="flex justify-between items-center max-w-7xl mx-auto h-full">


                    <div className="flex items-center gap-6">
                        <Link href="/" className="select-none cursor-pointer contents" scroll={false}>
                            <h1 className="text-[26px] font-bold text-primary-blue justify-self-start">Seer</h1>
                        </Link>
                        <div className="gap-6 items-center mt-1 hidden lg:flex">
                            <Link href="/markets" className="text-[13px] text-white hover:text-primary-blue transition-all duration-150 font-medium">Markets</Link>
                            <Link href="/leaderboard" className="text-[13px] text-white hover:text-primary-blue transition-all duration-150 font-medium">Leaderboard</Link>
                        </div>
                        <div className="w-150 shrink-0 hidden lg:block">
                            <SearchBar />
                        </div>

                    </div>


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

        </>
    );
}
