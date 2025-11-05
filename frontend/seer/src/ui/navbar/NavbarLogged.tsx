import { useUserQuery } from "@/lib/queries/useUserQuery";
import { useDrawerStore } from "@/lib/stores/drawer";
import { Bell, MessageCircleMore, Search } from "lucide-react";
import Link from "next/link";
import BetSlip from "../../../public/icons/bet_slip2.svg";
import Balance from "../Balance";
import ToolTip from "../ToolTip";
import UserPart from "./UserPart";

export default function NavbarLogged() {


    const user = useUserQuery().data;
    const openDrawer = useDrawerStore((state) => state.openDrawer);
    if (!user) return null;



    return (
        <>
            <div className="justify-self-center">
                <Balance currency="USDT" />
            </div>

            <ul className="flex gap-2.5 justify-self-end">

                <li className="hidden lg:block">
                    <Link className="contents" href="/mybets">
                        <ToolTip Icon={BetSlip} bgFull />
                    </Link>

                </li>

                <li className="hidden lg:block">
                    <ToolTip Icon={Bell} bgFull />
                </li>

                <li className="hidden lg:block">
                    <ToolTip Icon={Search} bgFull />
                </li>

                <li className="hidden lg:block" onClick={() => openDrawer("chat")}>
                    <ToolTip Icon={MessageCircleMore} bgFull />
                </li>

                <UserPart />

            </ul>
        </>
    )
}
