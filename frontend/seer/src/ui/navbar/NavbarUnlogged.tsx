import { useModalStore } from "@/lib/stores/modal";
import Button from "../Button";

export default function NavbarUnlogged() {
    const openModal = useModalStore((state) => state.openModal);
    return (
        <div className="flex gap-2">
            <Button bg="bg-gray-special" width="small" onClick={() => openModal("auth", { selectedTab: "login" })}>
                Login
            </Button>
            <Button bg="bg-neon-blue" width="small" onClick={() => openModal("auth", { selectedTab: "register" })}>
                Register
            </Button>
        </div>
    )
}