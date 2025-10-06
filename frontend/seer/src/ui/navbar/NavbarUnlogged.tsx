import Button from "../Button";
import { useModal } from "../modal/Modal";

export default function NavbarUnlogged() {
    const { openModal } = useModal();
    return (
        <div className="flex gap-2">
            <Button bg="bg-gray-special" width="small" onClick={() => openModal("auth", "login")}>
                Login
            </Button>
            <Button bg="bg-neon-blue" width="small" onClick={() => openModal("auth", "register")}>
                Register
            </Button>
        </div>
    )
}