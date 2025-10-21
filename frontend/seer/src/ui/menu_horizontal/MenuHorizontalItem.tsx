
export default function MenuHorizontalItem({ active = false, children, ...rest }: { active?: boolean, children: React.ReactNode, } & React.HTMLAttributes<HTMLButtonElement>) {
    return (
        <button
            className={`
        h-9.5 
        flex
        flex-1

        sm:flex-none
        justify-center
        font-medium 
        select-none 
        items-center 
        text-sm px-3
        ${!active && " "}
        hover:text-white
        rounded-md 
        transition-all 
        ${active ? "bg-gray-600 text-white" : "text-gray-200 active:scale-95 hover:bg-primary-blue active:bg-blue-pressed cursor-pointer"}
        duration-200
        `}
            {...rest}>
            {children}
        </button>
    )
}