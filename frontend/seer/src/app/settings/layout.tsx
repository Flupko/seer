"use client";

import MainWrapper from "@/ui/MainWrapper";
import MenuHorizontal from "@/ui/menu_horizontal/MenuHorizontal";
import MenuHorizontalItem from "@/ui/menu_horizontal/MenuHorizontalItem";
import Link from "next/link";

import { useSelectedLayoutSegment } from 'next/navigation'

export default function SettingsLayout({ children }: { children: React.ReactNode }) {

    const active = useSelectedLayoutSegment()

    return (
        <MainWrapper>
            <section className="min-h-screen bg-black text-white">
                <h1 className="text-2xl font-bold mb-6 text-white">Settings</h1>

                <MenuHorizontal>
                    <Link href="profile"> <MenuHorizontalItem active={active == "profile"}>Profile</MenuHorizontalItem></Link>
                    <Link href="security"><MenuHorizontalItem active={active == "security"}>Security</MenuHorizontalItem></Link>
                    <Link href="preferences"><MenuHorizontalItem active={active == "preferences"}>Preferences</MenuHorizontalItem></Link>
                    <Link href="verification"><MenuHorizontalItem active={active == "verification"}>Verification</MenuHorizontalItem></Link>
                </MenuHorizontal>

                <div className="mt-6">
                    {children}
                </div>
            </section>
        </MainWrapper >
    )
}