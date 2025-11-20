
export type InputProps = {
    leftEl?: React.ReactNode;
    rightEl?: React.ReactNode;
    className?: string;
    hasError?: boolean;
    bg?: string;
    border?: string;
} & React.InputHTMLAttributes<HTMLInputElement>;

export default function Input({ leftEl, rightEl, hasError, className, bg = "bg-gray-800", border = "border-gray-700", ...rest }: InputProps) {


    return (
        <div className="flex justify-between items-start relative w-full gap-1">
            {leftEl && <span className="flex justify-center items-center absolute h-full left-3">{leftEl}</span>}
            <input  {...rest}

                className={`${className} input-base w-full ${bg} flex-nowrap rounded-lg text-sm h-12 py-3 
                ${leftEl ? "pl-9" : "pl-4"} 
                ${rightEl ? "pr-11" : "pr-3"} 
                disabled:placeholder:text-gray-500
                border-[1px] outline-none focus:border-primary-blue ${hasError ? "border-red-500" : border} placeholder:text-gray-500 text-white`} />
            {rightEl && <span className="flex justify-center items-center absolute h-full right-3">{rightEl}</span>}
        </div>
    )
}



// export default function Input({ type, placeholder, value, onChange, leftEl, rightEl }: { type: string, placeholder: string, label?: string, value: string, onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void, leftEl?: React.ReactNode, rightEl?: React.ReactNode, error?: string }) {

//     return (
//         <div>
//             {label && <label className="text-xs font-semibold text-white mb-1 block">{label}</label>}
//             <div className="flex justify-between items-start relative w-full gap-1">
//                 {leftEl && <span className="flex justify-center items-center absolute h-full left-3">{leftEl}</span>}
//                 <input type={type} placeholder={placeholder} onChange={onChange}
//                     className={`w-full bg-gray-800 flex-nowrap rounded-lg text-sm h-12 py-3 ${leftEl ? "pl-9" : "pl-4"} ${rightEl ? "pr-11" : "pr-3"} border-[1px] outline-none focus:border-main-blue ${error ? "border-red-500" : "border-transparent"} placeholder:text-gray-400 text-white`} />
//                 {rightEl && <span className="flex justify-center items-center absolute h-full right-3">{rightEl}</span>}
//             </div>
//             {error && <label className="text-xs font-semibold text-red-500 mt-2 block">{error}</label>}
//         </div>

//     )
// }