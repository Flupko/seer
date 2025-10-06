import { Headphones, LogOut, Menu, Settings, Trophy, UserRound, Wallet } from "lucide-react"
import { AnimatePresence } from "motion/react"
import { useState } from "react"
import MenuLarge from "../menu_large_vertical/MenuLarge"
import MenuLargeItem from "../menu_large_vertical/MenuLargeItem"
import ToolTip from "../ToolTip"
import { div } from "motion/react-client"
import { useUserQuery } from "@/lib/queries/useUserQuery"
import Image from "next/image"
import * as api from "@/lib/api"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useRouter } from "next/navigation"
import Link from "next/link"
import ProfilePicture from "../ProfilePicture"


export default function MenuLogged() {

    const [menuClicked, setMenuClicked] = useState(false)
    const [menuHovering, setMenuHovering] = useState(false)

    return (
        <li
            className="relative"
            onMouseEnter={() => setMenuHovering(true)}
            onMouseLeave={() => setMenuHovering(false)}
        >
            <ToolTip Icon={UserRound} bgFull onClick={() => {
                setMenuClicked((prev) => {
                    if (prev == true) {
                        setMenuHovering(false)
                    }
                    return !prev
                })
            }} />

            <AnimatePresence>
                {(menuHovering || menuClicked) && (
                    <div className="absolute top-9 py-2.5 right-0">
                        <MenuLarge>

                            <UserInfo />

                            <MenuItem Icon={Wallet} text="Wallet" />
                            <MenuItem Icon={Trophy} text="Leaderboard" />
                            <Link href={"/settings/profile"}>
                                <MenuItem Icon={Settings} text="Settings" />
                            </Link>
                            <MenuItem Icon={Headphones} text="Support" />
                            <Logout />


                        </MenuLarge>
                    </div>
                )}
            </AnimatePresence>
        </li>
    )
}

function MenuItem({ Icon, text, ...rest }: { Icon: React.ElementType, text: string } & React.HTMLAttributes<HTMLButtonElement>) {
    return (
        <MenuLargeItem height="h-12" {...rest}>
            <div className="flex items-center gap-2">
                <Icon strokeWidth={1.3} size={19} />
                {text}
            </div>
        </MenuLargeItem>
    )
}

function UserInfo() {

    const { data } = useUserQuery()
    const user = data
    if (!user) return null

    return (
        <MenuLargeItem height="h-14">

            <div className="flex items-center gap-2.5">
                <ProfilePicture url={user.profileImageUrl} size={30}/>
                <span className="font-semibold text-lg">{user.username}</span>
            </div>

        </MenuLargeItem>
    )
}

function Logout() {

    const router = useRouter();
    const queryClient = useQueryClient();

    const { mutate } = useMutation({
        mutationFn: api.logout,
        onSuccess: () => {

            queryClient.resetQueries(); // clear everything in the cache
            queryClient.setQueryData(['user'], null);
            router.push("/")
        },
        onError: (error: api.APIError) => {
            console.error("Logout failed:", error);
        },
    });



    return (
        <MenuItem Icon={LogOut} text="Logout" onClick={() => mutate()} />
    )
}