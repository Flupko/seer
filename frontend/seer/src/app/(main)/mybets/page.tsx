"use client";

import { getUserBets } from "@/lib/api";
import { UserBetSearch, UserBetSearchSchema, UserBetsRes } from "@/lib/definitions";
import { getNextPageParamFromMetadata } from "@/lib/meta";
import { betSearchKey } from "@/lib/queries/betSearchKey";
import BetUser from "@/ui/bet/BetUser";
import Heading from "@/ui/Heading";
import MainWrapper from "@/ui/MainWrapper";
import MenuVertical from "@/ui/menu_small_vertical/MenuVertical";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Ticket } from "lucide-react";
import { useState } from "react";



export default function HomePage() {

  const [selectedTab, setSelectedTab] = useState<"active" | "resolved">("active");

  const parsed = UserBetSearchSchema.safeParse({
    status: selectedTab,
    pageSize: 18,
    page: 1,
    sort: selectedTab === "active" ? "placedAt" : "event",
    sortDir: 'desc',
  });

  const search = parsed.data;

  console.log("search", search);
  console.log("parsed error", parsed.error);

  const {
    data,
    isLoading,
    isError,
    fetchNextPage,
    isFetchingNextPage,
    hasNextPage,
  } = useInfiniteQuery({
    queryKey: betSearchKey(search),
    queryFn: ({ pageParam = 1 }) => getUserBets({ ...search, page: pageParam } as UserBetSearch),
    getNextPageParam: (lastPage: UserBetsRes) => getNextPageParamFromMetadata(lastPage.metadata),
    initialPageParam: 1,
    staleTime: 5 * 60 * 1000,
    enabled: parsed.success,
  });

  const bets = data?.pages.flatMap((p) => p.bets)

  return (

    <>
      <MainWrapper>
        <div className="flex flex-col md:flex-row md:items-center justify-between mb-0.5 md:mb-5 gap-5">

          <div className="flex items-center gap-2 shrink-0">
            <Ticket className="w-6.5 h-6.5 text-primary-blue block mt-0.5" />
            <Heading>My Bets</Heading>
          </div>

          <div className="md:w-38">
            <MenuVertical choices={[{ element: "Active Bets", value: "active" }, { element: "Settled Bets", value: "resolved" }]}
              onChange={(v) => setSelectedTab(v)}
              value={selectedTab} />
          </div>


        </div>


      </MainWrapper>


      <div className="px-0 md:px-12">
        <div className="max-w-7xl mx-auto">
          <div className="flex flex-col gap-4 bg-transparent md:bg-gray-900 md:rounded-md py-4 px-4">
            {/* TODO Skeleton */}
            {isLoading && <div>Loading bets...</div>}
            {isError && <div>Error loading bets</div>}
            {!isLoading && !isError && bets && bets.length === 0 && (
              <div>No bets found.</div>
            )}


            {!isLoading && !isError && bets && bets.length > 0 && (
              <>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4.5" key={Math.random()}>
                  {bets.map(bet => (
                    <BetUser bet={bet} key={bet.id} />
                  ))}
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </>


  )


}
