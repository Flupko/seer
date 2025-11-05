import { Bell, Home, MessageCircleIcon, ReceiptText, Search } from "lucide-react"
import Link from "next/link"


export default function NavbarMobile() {
    return (
        <div className="sticky bottom-0 lg:hidden w-full flex items-center bg-gray-900 border-t border-t-gray-700 h-15 z-50 [&>*:not(:last-child)]:border-r-gray-700 [&>*:not(:last-child)]:border-r">

            <MobileElement text="Home" Icon={Home} />
            <MobileElement text="Chat" Icon={MessageCircleIcon} />
            <MobileElement text="Search" Icon={Search} />
            <MobileElement text="Notifs" Icon={Bell} />

            <Link className="contents" href="/mybets">
                <MobileElement text="My Bets" Icon={ReceiptText} />
            </Link>

        </div>
    )
}

function MobileElement({ text, Icon }: { text: string, Icon: React.ElementType } & React.HTMLAttributes<HTMLDivElement>) {
    return (
        <div className="flex flex-1 justify-center items-center h-full text-white">
            <button className="w-full active:scale-95 transition-all duration-200">
                <div className="flex flex-col items-center justify-center cursor-pointer font-bold text-[0.65rem]">
                    <Icon className="w-5 h-5 mt-1" strokeWidth={1.7} />
                    <span className="mt-[0.2rem]">{text}</span>
                </div>
            </button>
        </div>
    )
}