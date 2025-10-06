import { useUserQuery } from "@/lib/queries/useUserQuery";
import Balance from "../Balance";
import ToolTip from "../ToolTip";
import { Bell, MessageCircleMore, Search, Settings, Trophy, UserRound, Wallet } from "lucide-react";
import { useState } from "react";
import MenuLarge from "../menu_large_vertical/MenuLarge";
import MenuLargeItem from "../menu_large_vertical/MenuLargeItem";
import { AnimatePresence } from "motion/react";
import UserPart from "./UserPart";

export default function NavbarLogged() {


    const user = useUserQuery().data;
    if (!user) return null;

    return (
        <>
            <div className="justify-self-center">
                <Balance />
            </div>

            <ul className="flex gap-2.5 justify-self-end">
                <li>
                    <ToolTip Icon={Bell} bgFull />
                </li>

                <li>
                    <ToolTip Icon={Search} bgFull />
                </li>

                <li>
                    <ToolTip Icon={MessageCircleMore} bgFull />
                </li>

                <UserPart />

            </ul>
        </>
    )
}
