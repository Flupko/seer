"use client";

import { UserBetSearch } from "@/lib/definitions";
import { useBetsQuery } from "@/lib/queries/useBetsQuery";
import BetUser from "@/ui/bet/BetUser";
import Button from "@/ui/Button";
import Heading from "@/ui/Heading";
import MenuVertical from "@/ui/menu_small_vertical/MenuVertical";
import { useState } from "react";



export default function HomePage() {

  const [selectedTab, setSelectedTab] = useState<"active" | "resolved">("active");

  const search: UserBetSearch = {
    status: selectedTab,
    pageSize: 5,
    page: 1,
    sort: selectedTab === "active" ? "placedAt" : "event",
    sortDir: 'desc',
  };


  const {
    data,
    isLoading,
    isError,
    fetchNextPage,
    isFetchingNextPage,
    hasNextPage,
  } = useBetsQuery({ search });

  const bets = data?.pages.flatMap((p) => p.bets)

  return (


    <>
      <div className="flex flex-col md:flex-row md:items-center justify-between mb-5 gap-5">

        <div className="flex items-center gap-2 shrink-0">
          <Heading>My Bets</Heading>
        </div>

        <div className="md:w-38">
          <MenuVertical choices={[{ element: "Active Bets", value: "active" }, { element: "Settled Bets", value: "resolved" }]}
            onChange={(v) => setSelectedTab(v)}
            value={selectedTab} />
        </div>


      </div>

      <div className="h-10 items-center justify-between w-full text-[11px] hidden md:flex px-4 border-b border-gray-700 mb-4">
        <div className="flex-2">
          <span className="text-gray-400 font-bold tracking-wider">MARKET</span>
        </div>

        <div className="flex-1">
          <span className="text-gray-400 font-bold tracking-wider">STAKE</span>
        </div>

        <div className="flex-1">
          <span className="text-gray-400 font-bold tracking-wider">TO WIN</span>
        </div>

        <div className={`${selectedTab === "active" ? "flex-2" : "w-50"}`}>
          < span className="text-gray-400 font-bold tracking-wider">{selectedTab === "active" ? "CASHOUT VALUE" : "RESULT"}</span>
        </div>
      </div >


      <div className="flex flex-col gap-4">
        {/* TODO Skeleton */}
        {isLoading && <div>Loading bets...</div>}
        {isError && <div>Error loading bets</div>}
        {!isLoading && !isError && bets && bets.length === 0 && (
          <div>No bets found.</div>
        )}


        {!isLoading && !isError && bets && bets.length > 0 && (
          <>
            <div className="flex flex-col gap-2.5" key={Math.random()}>
              {bets.map(bet => (
                <BetUser bet={bet} key={bet.id} />
              ))}
            </div>

            {hasNextPage && (
              <div className="flex justify-center mt-4">
                <Button
                  bg="bg-primary-blue"
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                  width="small"
                >
                  {isFetchingNextPage
                    ? 'Loading more...'
                    : hasNextPage
                      ? 'Load More'
                      : 'No more bets'}
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </>


  )


}
