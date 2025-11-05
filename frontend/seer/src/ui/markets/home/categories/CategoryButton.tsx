import { Category } from "@/lib/definitions";
import Image from "next/image";
import { HTMLAttributes } from "react";

export default function CategoryButton({ category, active = false, ...rest }: { category: Category, active?: boolean } & HTMLAttributes<HTMLDivElement>) {
    return (
        <div
            className={`snap-start w-14 flex flex-col items-center justify-center gap-1.5 select-none ${active ? "mb-0" : "cursor-pointer"} hover:mb-0 transition-all duration-300 ${active ? "" : "hover:[&>div:first-child]:bg-primary-blue active:[&>div:first-child]:scale-95"}`}
            {...rest}
        >
            <div
                className={`w-13.5 h-13.5 bg-gray-800 rounded-lg border flex items-center justify-center transition-all duration-200
                ${active ? "border-neon-blue" : "border-transparent"}`}>
                <Image src={category.iconUrl} alt={category.label} width={30} height={30} className={`${active ? "filter-(--primary-blue-filter)" : ""}`} />
            </div>
            <div className={`${active ? "text-primary-blue font-bold" : "text-gray-400 font-medium"} text-xs text-ellipsis max-w-full text-center line-clamp-1 text-wrap break-after-auto`}>{category.label}</div>
        </div>
    )
}