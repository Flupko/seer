
export function TableHeading({ children, className }: { children: React.ReactNode, className?: string }) {
    return (
        <th className={`text-gray-400 text-sm text-left font-normal px-6 text-nowrap ${className}`}>
            {children}
        </th>
    )
}

export function TableRow({ children, className }: { children: React.ReactNode, className?: string }) {

    return (
        <tr className={`rounded-xl text-white transition-all`}>
            {children}
        </tr>
    )
}

export function TableCell({ children, current, className }: { children: React.ReactNode, current: boolean, className?: string }) {
    return (
        <td className={`text-white text-left font-medium text-sm ${current ? "bg-gray-900" : "bg-grayscale-black"} px-6 py-4.5 ${className}`}>
            {children}
        </td>
    )
}

export function TableHead({ children, className }: { children: React.ReactNode, className?: string }) {
    return (
        <thead className={`h-12 border-separate border-spacing-0 ${className}`}>
            <tr className="text-gray-200 text-sm text-left">
                {children}
            </tr>
        </thead>
    )
}