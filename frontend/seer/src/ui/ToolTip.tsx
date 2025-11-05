

export default function ToolTip({ Icon, bgFull = false, ...rest }: { Icon: React.ComponentType<any>, bgFull?: boolean } & React.ButtonHTMLAttributes<HTMLDivElement>) {
    return (
        <div
            {...rest}
            className={`w-10 h-10 active:scale-95 rounded-full ${bgFull ? "bg-gray-700 hover:bg-gray-600 active:bg-gray-500 transition-all duration-200" : "hover:bg-gray-700 [transition:background-color_330ms,transform_200ms]"} flex justify-center items-center cursor-pointer`}
        >
            <Icon className="text-white" size={19} strokeWidth={1.5} />
        </div>
    )
}