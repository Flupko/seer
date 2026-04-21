import * as api from "@/lib/api"
import { useUserQuery } from "@/lib/queries/useUserQuery"
import { useMutation } from "@tanstack/react-query"
import { Headphones, LogOut, Settings, Trophy, UserRound, Wallet } from "lucide-react"
import { AnimatePresence } from "motion/react"
import Link from "next/link"
import { useEffect, useRef, useState } from "react"
import MenuLarge from "../menu_large_vertical/MenuLarge"
import MenuLargeItem from "../menu_large_vertical/MenuLargeItem"
import ProfilePicture from "../ProfilePicture"
import ToolTip from "../ToolTip"


export default function MenuLogged() {

    const [menuClicked, setMenuClicked] = useState(false)
    const [menuHovering, setMenuHovering] = useState(false)

    const wrapRef = useRef<HTMLLIElement | null>(null);

    const closeMenu = () => {
        setMenuClicked(false)
        setMenuHovering(false)
    }

    useEffect(() => {
        const onDown = (e: MouseEvent | PointerEvent) => {
            if (!wrapRef.current) return;
            if (!wrapRef.current.contains(e.target as Node)) closeMenu();
        };
        const onKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") closeMenu();
        };
        document.addEventListener("mousedown", onDown);
        document.addEventListener("pointerdown", onDown);
        document.addEventListener("keydown", onKey);
        return () => {
            document.removeEventListener("mousedown", onDown);
            document.removeEventListener("pointerdown", onDown);
            document.removeEventListener("keydown", onKey);
        };
    }, []);

    return (
        <div
            ref={wrapRef}
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

                            <UserInfo onClick={closeMenu} />

                            <MenuItem Icon={Wallet} text="Wallet" onClick={closeMenu} />
                            <MenuItem Icon={Trophy} text="Leaderboard" onClick={closeMenu} />
                            <Link href={"/settings/profile"}>
                                <MenuItem Icon={Settings} text="Settings" onClick={closeMenu} />
                            </Link>
                            <MenuItem Icon={Headphones} text="Support" onClick={closeMenu} />
                            <Logout />


                        </MenuLarge>
                    </div>
                )}
            </AnimatePresence>
        </div>
    )
}

function MenuItem({ Icon, text, ...rest }: { Icon: React.ElementType, text: string } & React.HTMLAttributes<HTMLButtonElement>) {
    return (
        <MenuLargeItem height="h-12" {...rest}>
            <div className="flex items-center gap-2.5">
                <Icon strokeWidth={1.3} size={20} />
                <span className="pb-0.5 text-sm font-medium">{text}</span>
            </div>
        </MenuLargeItem>
    )
}

function UserInfo({ ...rest }: React.HTMLAttributes<HTMLButtonElement>) {

    const { data: user } = useUserQuery()
    if (!user) return null

    return (
        <MenuLargeItem height="h-14" {...rest}>

            <div className="flex items-center gap-2.5">
                <ProfilePicture url={user.profileImageUrl} size={30} />
                <span className="font-semibold text-lg">{user.username}</span>
            </div>

        </MenuLargeItem>
    )
}

function Logout() {

    const { mutate } = useMutation({
        mutationFn: api.logout,
        onSuccess: () => {
            window.location.replace('/');
        },
        onError: (error: api.APIError) => {
            console.error("Logout failed:", error);
        },
    });



    return (
        <MenuItem Icon={LogOut} text="Logout" onClick={() => mutate()} />
    )
}