

export function OutcomeBadge({ children, className, smaller = false }: { children: React.ReactNode, className?: string, smaller?: boolean }) {
    return (
        <div
            className={`${smaller ? "text-xs h-5.5" : "text-sm h-6"}  shrink-0 font-bold flex items-center justify-center text-white rounded-sm px-1.5 w-fit ${className ?? ""}`}>
            {children}
        </div>
    )
}