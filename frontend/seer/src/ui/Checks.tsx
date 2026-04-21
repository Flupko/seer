import { Check, ClockFading, X } from "lucide-react";


export function CheckNo({ size = "w-2.5 h-2.5", padding = "p-[0.2rem]", className }: { size?: string, padding?: string, className?: string }) {
    return (
        <div className={`rounded-full ${padding} ${className} flex items-center justify-center`}>
            <X className={`${size}`} strokeWidth={4} />
        </div>
    )
}

export function CheckYes({ size = "w-2.5 h-2.5", padding = "p-[0.2rem]", className }: { size?: string, padding?: string, className?: string }) {
    return (
        <div className={`rounded-full ${padding} ${className} flex items-center justify-center`}>
            <Check className={`${size} translate-y-[0.5px]`} strokeWidth={4} />
        </div>
    )
}

export function Time({ size = "w-2.5 h-2.5", padding = "p-[0.2rem]", className }: { size?: string, padding?: string, className?: string }) {
    return (
        <div className={`rounded-full ${padding} ${className} flex items-center justify-center`}>
            <ClockFading className={`${size}`} strokeWidth={2.5} />
        </div>
    )
}