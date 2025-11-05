import { useDrawerStore } from "@/lib/stores/drawer";
import { X } from "lucide-react";



export default function DrawerHeader({ title, left, right }: { title: string, left?: React.ReactNode, right?: React.ReactNode }) {

    const closeDrawer = useDrawerStore((state) => state.closeDrawer)

    return (
        <div className="flex justify-between items-center bg-gray-900 px-5 border-b border-b-gray-700 h-19 shrink-0">
            <div className="h-full flex items-center gap-2">
                <h2 className="text-xl font-bold">{title}</h2>
                {left}
            </div>

            <div className="h-full flex items-center gap-2">
                {right}
                <div className="h-full flex items-center cursor-pointer group" onClick={closeDrawer}>
                    <X className="w-5 h-5 group-hover:text-neon-blue transition-colors" strokeWidth={1.8} />
                </div>

            </div>

        </div>
    )
}