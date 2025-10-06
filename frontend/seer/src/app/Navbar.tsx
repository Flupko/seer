"use client"

import Button from "@/ui/Button";
import { useModal } from "@/ui/modal/Modal";
import { useUserQuery } from "@/lib/queries/useUserQuery";
import NavbarUnlogged from "@/ui/navbar/NavbarUnlogged";
import NavbarLogged from "@/ui/navbar/NavbarLogged";
import { useEffect } from "react";
import { div } from "motion/react-client";

export default function Navbar() {
    const { data: user, isPending } = useUserQuery();

    return (
        <nav className="border-b border-gray-700 sticky px-4 md:px-12">
            <div className="grid grid-cols-3 items-center h-19 max-w-7xl mx-auto">
                <h1 className="text-3xl font-bold text-white justify-self-start">Seer</h1>

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
