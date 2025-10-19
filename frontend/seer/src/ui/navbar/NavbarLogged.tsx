import { useUserQuery } from "@/lib/queries/useUserQuery";
import { useDrawerStore } from "@/lib/stores/drawer";
import { Bell, MessageCircleMore, Search } from "lucide-react";
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
                <Balance />
            </div>

            <ul className="flex gap-2.5 justify-self-end">

                <li>
                    <ToolTip Icon={Bell} bgFull />
                </li>

                <li>
                    <ToolTip Icon={Search} bgFull />
                </li>

                <li onClick={() => openDrawer("chat")}>
                    <ToolTip Icon={MessageCircleMore} bgFull />
                </li>

                <UserPart />

            </ul>
        </>
    )
}
