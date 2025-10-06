
export default function MenuHorizontalItem({ active = false, children, ...rest }: { active?: boolean, children: React.ReactNode, } & React.HTMLAttributes<HTMLButtonElement>) {
    return (
        <button
            className={`
        h-9.5 
        flex
        flex-1
        md:flex-none
        justify-center
        font-medium 
        select-none 
        items-center 
        text-sm px-3
        ${!active && " "}
        hover:text-white
        rounded-md 
        transition-all 
        ${active ? "bg-gray-600 text-white" : "text-gray-100 active:scale-90 active:brightness-70 hover:bg-gray-500 cursor-pointer"}
        duration-200
        `}
            {...rest}>
            {children}
        </button>
    )
}