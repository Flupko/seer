"use client";

import { MarketSearchRes, searchMarket } from "@/lib/api";
import { Category, MarketSearch, MarketSearchSchema } from "@/lib/definitions";
import { getNextPageParamFromMetadata } from "@/lib/meta";
import { marketSearchKey } from "@/lib/queries/marketSearchKey";
import Button from "@/ui/Button";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useSearchParams } from "next/navigation";
import MarketCardBis from "./MarketCardBis";


export default function MarketsDisplay({ categories }: { categories: Category[] }) {

    const sp = useSearchParams();

    const categorySlug = sp.get("category") ?? categories[0].slug;
    const sort = sp.get("sort") ?? "trending";
    const status = sp.get("status") ?? "active";

    const parsed = MarketSearchSchema.safeParse({
        categorySlug,
        sort,
        status,
        pageSize: 6,
        page: 1,
    });

    const search = parsed.data;

    console.log("Market search params:", search);

    const {
        data,
        isLoading,
        isError,
        fetchNextPage,
        isFetchingNextPage,
        hasNextPage,
    } = useInfiniteQuery({
        queryKey: marketSearchKey(search),
        queryFn: ({ pageParam = 1 }) => searchMarket({ ...search, page: pageParam } as MarketSearch),
        getNextPageParam: (lastPage: MarketSearchRes) => getNextPageParamFromMetadata(lastPage.metadata),
        initialPageParam: 1,
        staleTime: 5 * 60 * 1000,
        enabled: parsed.success,
    });

    const markets = data?.pages.flatMap((p) => p.markets)

    return (

        <>
            <div className="flex flex-col gap-4 md:bg-gray-900 rounded-md md:py-4 lg:px-4 ">

                {/* TODO Skeleton */}
                {isLoading && <div>Loading markets...</div>}
                {isError && <div>Error loading markets</div>}
                {!isLoading && !isError && markets && markets.length === 0 && (
                    <div>No markets found.</div>
                )}


                {!isLoading && !isError && markets && markets.length > 0 && (
                    <>
                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4.5">
                            {markets.map(market => (
                                <MarketCardBis marketInitial={market} key={market.id} />
                            ))}
                        </div>
                    </>
                )}
            </div>
            {hasNextPage &&
                <div className="flex justify-center mt-4">

                    <div className="w-full max-w-[16rem]">
                        <Button
                            bg="bg-neon-blue"
                            width="full"
                            onClick={() => fetchNextPage()}
                            className="px-4 py-2 text-white font-bold rounded-md transition duration-200 cursor-pointer select-none text-sm"
                        >
                            {isFetchingNextPage ? "Loading more..." : "Load More"}
                        </Button>
                    </div>

                </div>}
        </>
    )
}