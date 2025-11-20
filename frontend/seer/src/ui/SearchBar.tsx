"use client";

import { getMarketById, searchMarket } from "@/lib/api";
import { MarketSearch, MarketView } from "@/lib/definitions";
import { useUpdateSearchParams } from "@/lib/hooks/useUpdateSearchParams";
import { marketSearchKey } from "@/lib/queries/marketSearchKey";
import { usePrefs } from "@/lib/stores/prefs";
import { useQuery } from "@tanstack/react-query";
import { SearchIcon, X } from "lucide-react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useDebounceCallback } from "usehooks-ts";
import { AnimatedOdds } from "./odds/AnimatedOdds";

export default function SearchBar() {

    const { setParams } = useUpdateSearchParams();

    const [inputValue, setInputValue] = useState("");
    const [open, setOpen] = useState(false);

    const searchArea = useRef<HTMLDivElement | null>(null);
    const inputRef = useRef<HTMLInputElement | null>(null);

    useEffect(() => {
        const onDown = (e: MouseEvent | PointerEvent) => {
            if (!searchArea.current) return;
            if (!searchArea.current.contains(e.target as Node)) setOpen(false);
        };
        const onKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") setOpen(false);
        };
        document.addEventListener("mousedown", onDown);
        document.addEventListener("pointerdown", onDown);
        document.addEventListener("keydown", onKey);
        return () => {
            document.removeEventListener("mousedown", onDown);
            document.removeEventListener("pointerdown", onDown);
            document.removeEventListener("keydown", onKey);
        };
    }, []);

    const [query, setQuery] = useState("");

    const handleChangeInput = (v: string) => {
        setInputValue(v)
        debouncedQuery(v)
    }

    const handleSearch = () => {
        setParams({ q: inputValue }, { newPathname: "/search", replace: true, overwriteCurrent: true })
        inputRef.current?.blur();
        setOpen(false);
    }


    const search: MarketSearch = {
        query: query,
        sort: "trending",
        page: 1,
        pageSize: 5,
    };

    const { data, isLoading, isError } = useQuery({
        queryKey: marketSearchKey(search),
        queryFn: () => searchMarket(search),
        enabled: query.length >= 3,
    });

    const debouncedQuery = useDebounceCallback(setQuery, 300);

    return (

        <div className="relative w-full" ref={searchArea}>
            <div className="flex justify-between items-start gap-1 relative">
                <span className="flex justify-center items-center absolute h-full left-3.5 cursor-pointer" onClick={handleSearch}>
                    <SearchIcon size={18} className="text-gray-400" strokeWidth={2.5} />
                </span>
                <input
                    ref={inputRef}
                    maxLength={50}
                    onKeyDown={(e) => {
                        if (e.key === "Enter") {
                            handleSearch();
                        }
                    }}
                    onFocus={() => setOpen(true)}
                    type="text"
                    value={inputValue}
                    onChange={e => handleChangeInput(e.target.value)}
                    className={`input-base w-full bg-gray-800 flex-nowrap rounded-lg text-sm font-normal h-10 py-3 pl-11 disabled:placeholder:text-gray-500 hover:bg-gray-700/80 focus:bg-transparent 
                border-1 border-transparent outline-none focus:border-primary-blue placeholder:text-gray-400 text-white transition-colors`} placeholder="Search for markets" />


                {inputValue &&
                    <span className="flex justify-center items-center absolute h-full right-3.5 cursor-pointer transition-all" onClick={() => {
                        setInputValue("")
                        setQuery("")
                    }}>
                        <X size={18} className="text-gray-400" />
                    </span>
                }

            </div >

            {/* Search results */}
            {(data?.markets?.length ?? 0) > 0 && open &&
                <div className="absolute top-9 pt-3 w-full z-10">
                    <div className="bg-grayscale-black rounded-lg px-2 py-2 flex flex-col gap-2 border border-gray-700 shadow-2xl">
                        {
                            data?.markets.map(market => <SearchResultMarket marketInitial={market} key={market.id} />)
                        }
                    </div>
                </div>

            }

        </div>


    )
}


function SearchResultMarket({ marketInitial }: { marketInitial: MarketView }) {
    const { data: market } = useQuery({
        queryKey: ['market', marketInitial.id],
        queryFn: () => getMarketById(marketInitial.id),
        initialData: marketInitial,
        staleTime: Infinity,
    })

    const topOutcome = market.outcomes.reduce((o1, o2) => o1.priceYes > o2.priceYes ? o1 : o2)
    const oddsFormat = usePrefs(state => state.oddsFormat)

    const router = useRouter();

    return (
        <div key={market.id} className="flex cursor-pointer justify-between rounded-md hover:bg-gray-800/50 p-1.5" onClick={() => router.push(`/market/${market.id}`)}>

            <div className="flex gap-4 items-center">
                {market.imgKey && (
                    <div className="flex-shrink-0 w-12 h-12 rounded-lg overflow-hidden bg-gray-900">
                        <Image
                            src={market.imgKey}
                            alt={market.name}
                            width={250}
                            height={250}
                            className="object-cover w-full h-full"
                        />
                    </div>
                )}
                <span className="text-[15px] font-bold">{market.name}</span>
            </div>

            {/* Most likely outcome */}
            <div className="flex flex-col justify-center text-right">
                <span className="text-base font-bold text-white"> <AnimatedOdds prob={topOutcome.priceYes} format={oddsFormat} /></span>
                <span className="text-xs font-medium text-gray-400 line-clamp-1">{topOutcome.name}</span>
            </div>

        </div>
    )
}