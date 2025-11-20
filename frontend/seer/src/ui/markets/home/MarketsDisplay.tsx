"use client";

import { MarketView } from "@/lib/definitions";
import Button from "@/ui/Button";
import { useInfiniteQuery } from "@tanstack/react-query";
import MarketCardBis from "./MarketCard";


export default function MarketsDisplay({ markets, isLoading, isError, fetchNextPage, isFetchingNextPage, hasNextPage }: { markets?: MarketView[] } & Partial<ReturnType<typeof useInfiniteQuery>>) {

    console.log(markets);

    return (

        <>
            <div className="flex flex-col gap-4 rounded-md">

                {/* TODO Skeleton */}
                {isLoading && <div>Loading markets...</div>}
                {isError && <div>Error loading markets</div>}
                {!isLoading && !isError && markets && markets.length === 0 && (
                    <div>No markets found.</div>
                )}


                {!isLoading && !isError && markets && markets.length > 0 && (
                    <>
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3.5">
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
                            onClick={() => fetchNextPage?.()}
                            className="px-4 py-2 text-white font-bold rounded-md transition duration-200 cursor-pointer select-none text-sm"
                        >
                            {isFetchingNextPage ? "Loading more..." : "Load More"}
                        </Button>
                    </div>

                </div>}
        </>
    )
}



const GridItemWithBorders = ({
    children,
    index,
    total,
    colsLg
}: {
    children: React.ReactNode;
    index: number;
    total: number;
    colsLg: number;
}) => {
    // Determine responsive columns
    const colsSm = 2;
    const colsMd = colsLg;

    // For lg screens (4 columns)
    const isNotLastColLg = (index + 1) % colsMd !== 0;
    const isNotLastRowLg = index < total - colsMd;

    const isFistCol = index % colsMd === 0;
    const isLastCol = (index + 1) % colsMd === 0;

    return (
        <div className={`relative px-6 ${isFistCol ? "pl-0" : ""} ${isLastCol ? "pr-0" : ""}`}>
            {children}

            {/* Vertical border - 80% height */}
            {isNotLastColLg && (
                <div
                    className={`absolute right-0 top-1/2 w-px bg-gray-800/80 transform -translate-y-1/2 h-[85%]`}
                />
            )}

            {/* Horizontal border - 90% width */}
            {isNotLastRowLg && (
                <div
                    className={`absolute bottom-0 left-1/2 h-px bg-gray-800/80 transform -translate-x-1/2 w-[90%]`}
                />
            )}
        </div>
    );
};