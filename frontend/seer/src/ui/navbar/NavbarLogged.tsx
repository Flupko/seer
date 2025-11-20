import { useUserQuery } from "@/lib/queries/useUserQuery";
import { useDrawerStore } from "@/lib/stores/drawer";
import { Bell, MessageCircleMore } from "lucide-react";
import Balance from "../Balance";
import UserPart from "./UserPart";

export default function NavbarLogged() {


    const user = useUserQuery().data;
    const openDrawer = useDrawerStore((state) => state.openDrawer);
    if (!user) return null;



    return (
        <div className="flex">


            <Balance currency="USDT" />

            <ul className="flex gap-3 items-center ml-2">

                <div className="flex gap-4">
                    {/* <div className="hidden lg:block">
                        <BetSlip className="w-5.5 h-5.5 cursor-pointer hover:brightness-120 transition-all duration-150" />
                    </div> */}

                    <div className="hidden lg:block">
                        <Bell className="w-5.5 h-5.5 cursor-pointer hover:brightness-120 transition-all duration-150 text-gray-400" />
                    </div>

                    <div className="hidden lg:block" onClick={() => openDrawer("chat")}>
                        <MessageCircleMore className="w-5.5 h-5.5 cursor-pointer hover:brightness-120 transition-all duration-150 text-gray-400" />
                    </div>
                </div>

                <div className="w-[1px] bg-gray-600 h-5 mx-1"></div>

                <UserPart />

            </ul>
        </div>

    )
}
