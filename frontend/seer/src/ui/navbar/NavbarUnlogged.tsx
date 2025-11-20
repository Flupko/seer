import { useModalStore } from "@/lib/stores/modal";

export default function NavbarUnlogged() {
    const openModal = useModalStore((state) => state.openModal);
    return (
        <div className="flex gap-2 items-center">

            <button
                className="bg-gray-800 px-3 font-medium rounded-lg h-10 text-[13px] shrink-0 hover:brightness-120 hover:cursor-pointer transition-all active:scale-95 duration-150"
                onClick={() => openModal("auth", { selectedTab: "login" })}>
                Log in
            </button>
            <button
                className="bg-primary-blue px-3 font-medium rounded-lg h-10 text-[13px] shrink-0 hover:brightness-120 hover:cursor-pointer transition-all active:scale-95 duration-150"
                onClick={() => openModal("auth", { selectedTab: "register" })}>
                Sign up
            </button>
        </div>
    )
}