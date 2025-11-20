
export default function MenuHorizontal({ children, ...rest }: { children: React.ReactNode } & React.HTMLAttributes<HTMLDivElement>) {
    return (
        <div className="bg-gray-800 px-1.5 pt-1.5 rounded-full sm:w-fit max-w-full" {...rest} >
            <div className="max-w-full overflow-x-auto flex items-center gap-1.5 pb-1.5">
                {children}
            </div>
        </div>
    )
}