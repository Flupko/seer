"use client";
export const expexperimental_ppr = true

import { getSessions, getUserPreferences } from "@/lib/api";
import MainWrapper from "@/ui/MainWrapper";
import MenuHorizontal from "@/ui/menu_horizontal/MenuHorizontal";
import MenuHorizontalItem from "@/ui/menu_horizontal/MenuHorizontalItem";
import PrefetchLink from "@/ui/PrefetchLink";
import Link from "next/link";

import { useSelectedLayoutSegment } from 'next/navigation';

export default function SettingsLayout({ children }: { children: React.ReactNode }) {

    const active = useSelectedLayoutSegment()

    return (
        <MainWrapper>
            <section className="min-h-screen text-white">
                <h1 className="text-2xl font-extrabold mb-6 text-white">Settings</h1>

                <MenuHorizontal>

                    <Link href="profile" className="contents">
                        <MenuHorizontalItem active={active == "profile"}>Profile</MenuHorizontalItem>
                    </Link>

                    <PrefetchLink href="security"
                        queries={[{ queryKey: ["user", "sessions", "showInactive", false], queryFn: () => getSessions(false) }]}>
                        <MenuHorizontalItem active={active == "security"}>Security</MenuHorizontalItem>
                    </PrefetchLink>

                    <PrefetchLink href="preferences"
                        queries={[{ queryKey: ["user", "preferences"], queryFn: () => getUserPreferences() }]}>
                        <MenuHorizontalItem active={active == "preferences"}>Preferences</MenuHorizontalItem>
                    </PrefetchLink>

                    <Link href="verification" className="contents">
                        <MenuHorizontalItem active={active == "verification"}>Verification</MenuHorizontalItem>
                    </Link>
                </MenuHorizontal>

                <div className="mt-6">
                    {children}
                </div>
            </section>
        </MainWrapper >
    )
}