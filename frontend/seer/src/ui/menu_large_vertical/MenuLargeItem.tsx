export default function MenuLargeItem({children, height, ...rest} : {children: React.ReactNode, height:string} & React.HTMLAttributes<HTMLButtonElement>) {
    return (
        <button className={`w-full flex select-none items-center cursor-pointer text-sm font-medium px-3 hover:bg-gray-800 rounded-md transition-all duration-200 active:scale-95 ${height}`} {...rest}>
            {children}
        </button>
    )
}