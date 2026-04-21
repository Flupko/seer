"use client";

import { Category, sortOptions, statusOptions } from "@/lib/definitions";
import { useUpdateSearchParams } from "@/lib/hooks/useUpdateSearchParams";
import MenuVertical from "@/ui/menu_small_vertical/MenuVertical";
import { SearchIcon, X } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { useState } from "react";
import { useDebounceCallback } from "usehooks-ts";



export function Header({ activeCategory }: { activeCategory: Category }) {

    const params = useSearchParams();
    const sort = params.get("sort") ?? "trending";
    const status = params.get("status") ?? "active";

    const [inputValue, setInputValue] = useState(params.get("q") ?? "");
    const handleChangeInput = (v: string) => {
        setInputValue(v);
        if (v.length < 3) {
            setParams({ q: "" });
            return;
        }
        debouncedQuery(v)
    }


    const handleSearch = () => {
        setParams({ q: inputValue }, { newPathname: "/search", replace: true, overwriteCurrent: true })
    }

    const debouncedQuery = useDebounceCallback((v) => setParams({ q: v }), 300);

    const { setParams } = useUpdateSearchParams();

    return (
        <div className="mb-4">

            <div className="flex items-center gap-3">

                <div className="flex justify-between items-start gap-1 relative w-60">
                    <span className="flex justify-center items-center absolute h-full left-3.5 cursor-pointer" onClick={handleSearch}>
                        <SearchIcon size={18} className="text-gray-400" strokeWidth={2.5} />
                    </span>
                    <input

                        maxLength={50}
                        type="text"
                        value={inputValue}
                        onChange={e => handleChangeInput(e.target.value)}
                        className={`input-base w-full bg-gray-800 flex-nowrap rounded-lg text-sm font-normal h-10 py-3 pl-11 disabled:placeholder:text-gray-500 not-focus:hover:bg-gray-700/80 
                border-1 border-transparent outline-none placeholder:text-gray-400 text-white transition-colors`}
                        placeholder="Search" />


                    {inputValue &&
                        <span className="flex justify-center items-center absolute h-full right-3.5 cursor-pointer transition-all" onClick={() => {
                            setInputValue("")
                            setParams({ q: "" })
                        }}>
                            <X size={18} className="text-gray-400" />
                        </span>
                    }

                </div >

                <div className="w-[1px] bg-gray-700 h-6 mx-1 hidden lg:block"></div>

                {/* <div className="flex items-center gap-2">
                    <Image src={activeCategory.iconUrl} alt={activeCategory.label} width={26} height={26} className="filter-(--primary-blue-filter) w-5.5 h-5.5" />
                    <Heading>{activeCategory.label}</Heading>
                </div> */}

                <div className="flex items-center gap-3 overflow-x-auto">

                    <div className="shrink-0">
                        <MenuVertical leftPart={<span className="text-gray-400 mr-1.5">Status:</span>}
                            choices={statusOptions.map(({ value, element }) => ({ value, element: <span className="text-sm font-medium">{element}</span> }))}
                            value={status}
                            menuWidth={170}
                            onChange={(value) => setParams({ status: value })}
                            height="h-10"
                            padding="px-3" />
                    </div>

                    <div className="shrink-0">
                        <MenuVertical leftPart={<span className="text-gray-400 mr-1.5">Sort:</span>}
                            choices={sortOptions.map(({ value, element }) => ({ value, element: <span className="text-sm font-medium">{element}</span> }))}
                            value={sort}
                            menuWidth={180}
                            onChange={(value) => setParams({ sort: value })}
                            height="h-10"
                            padding="px-3" />
                    </div>

                </div>




            </div>


        </div>
    )
}