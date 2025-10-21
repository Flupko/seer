import Button from "../Button";
import DrawerHeader from "../drawer/DrawerHeader";
export default function BetSuccessDrawer(/* props */) {
    return (
        <>
            <DrawerHeader title="Place Bet" />

            {/* Drawer body: column, take remaining height below header */}

            <div className="flex flex-col h-[calc(100vh-10rem)] items-center justify-center min-h-0 overflow-hidden px-5">
                {/* Replace 56px with your header height */}
                <div className="flex flex-col gap-2 items-center w-full">
                    <div className="text-2xl font-semibold">Success</div>
                    <p className="text-gray-300 text-base">Your bet has been placed successfully.</p>
                    <div className="mt-4 w-full">
                        <Button bg="bg-neon-blue" width="full">
                            View Bets
                        </Button>
                    </div>
                </div>
            </div>

        </>
    );
}
