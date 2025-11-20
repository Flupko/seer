import { cashoutBet, getMarketById } from "@/lib/api";
import { Bet, MarketView, UserBetSearch } from "@/lib/definitions";
import { possiblePayoutDeltaForCashout } from "@/lib/lslmsr/lslmsr";
import { useBetsQuery } from "@/lib/queries/useBetsQuery";
import { useUserQuery } from "@/lib/queries/useUserQuery";
import Button from "@/ui/Button";
import { AnimatedOdds } from "@/ui/odds/AnimatedOdds";
import { toastStyled } from "@/ui/Toast";
import NumberFlow from "@number-flow/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { OutcomeBadge } from "../OutcomeBadge";


export default function MarketBetSection({ market }: { market: MarketView }) {

    const { data: user } = useUserQuery();

    if (!user) return null;

    const search: UserBetSearch = {
        status: "active",
        pageSize: 5,
        page: 1,
        sort: 'placedAt',
        sortDir: 'desc',
        marketId: market.id,
    }



    const [cashoutClicked, setCashoutClicked] = useState(false);

    const {
        data,
        isLoading,
        isError,
        fetchNextPage,
        isFetchingNextPage,
        hasNextPage,
    } = useBetsQuery({ search });

    return (
        <div className="mt-14">

            <h3 className="text-lg md:text-xl font-bold text-white mb-5">Active Bets</h3>

            <div className="border border-gray-800 rounded-xl p-4">


                <div className="h-10 items-center justify-between w-full text-[11px] hidden md:flex px-4">
                    <div className="flex-2">
                        <span className="text-gray-400 font-bold tracking-wider">OUTCOME</span>
                    </div>

                    {!market.isBinary && (<div className="flex-1">
                        <span className="text-gray-400 font-bold tracking-wider">SIDE</span>
                    </div>)}


                    <div className="flex-1">
                        <span className="text-gray-400 font-bold tracking-wider">STAKE</span>
                    </div>

                    <div className="flex-1">
                        <span className="text-gray-400 font-bold tracking-wider">TO WIN</span>
                    </div>

                    <div className="flex-2">
                        <span className="text-gray-400 font-bold tracking-wider">CASHOUT VALUE</span>
                    </div>
                </div>
                {/* 
                <div
                    className="shrink-0 self-stretch h-px w-full bg-gray-700 hidden sm:block"
                /> */}


                <div className="flex flex-col gap-2.5">
                    {isLoading && <div>Loading bets...</div>}

                    {data?.pages.flatMap((p) => p.bets).map((bet) => (
                        <BetCard bet={bet} key={bet.id} />
                    ))}
                </div>

                {hasNextPage && (
                    <div className="flex justify-center mt-3">
                        <button
                            onClick={() => fetchNextPage()}
                            disabled={isFetchingNextPage}
                            className="px-4 h-10 cursor-pointer bg-gray-800 hover:bg-gray-700 text-gray-300 hover:text-white rounded-lg text-sm font-medium active:scale-95 transition-all"
                        >
                            {isFetchingNextPage ? "Loading..." : "Load More"}
                        </button>
                    </div>
                )}

            </div>



        </div>
    );
}

export function BetCard({ bet }: { bet: Bet }) {

    const idempotencyKey = useRef(Math.random().toString());

    const { data: market } = useQuery({
        queryKey: ['market', bet.marketId],
        queryFn: () => getMarketById(bet.marketId),
        staleTime: Infinity,
    })

    const [cashoutClicked, setCashoutClicked] = useState(false);

    const queryClient = useQueryClient();

    const { mutate, isSuccess, isPending } = useMutation({
        mutationFn: cashoutBet,
        onSuccess: () => {
            // Invalidate 
            toastStyled("Bet cashed out successfully", { type: "success", autoClose: 1500 });
            queryClient.invalidateQueries({ queryKey: ['userBets'] });
        },
    });

    const handleCashout = () => {

        if (!cashoutClicked || !possible) return;

        mutate({
            betId: bet.id,
            minWantedGain: payout,
            idempotencyKey: idempotencyKey.current,
        });
    }

    const [possible, deltaProp, payout] = market ? possiblePayoutDeltaForCashout(market, bet) : [null, null, null];

    return (
        <>
            <div className="hidden md:flex items-center justify-between bg-gray-800/50 px-4 py-2 rounded-lg">
                <div className="flex-2 pl-0.5">
                    {!market?.isBinary && (<span className="text-sm font-bold text-white">{bet.outcomeName}</span>)}

                    {market?.isBinary && (
                        <OutcomeBadge className={`${market.outcomes[0].id === bet.outcomeId ? "bg-[#285cac]" : "bg-[#9a45fe]"}`}>{bet.outcomeName}</OutcomeBadge>
                    )}
                </div>

                {!market?.isBinary && (
                    <div className="flex-1">
                        <OutcomeBadge className={`${bet.side === "y" ? "bg-[#285cac]" : "bg-[#9a45fe]"}`}>{bet.side === "y" ? "Yes" : "No"}</OutcomeBadge>
                    </div>
                )}


                <div className="flex-1 pl-0.5">
                    <span className="text-sm font-bold text-white">${bet.pricePaid.toFixed(2)}</span>
                </div>
                <div className="flex-1 pl-0.5">
                    <span className="text-sm font-bold text-white">${bet.payout.toFixed(2)}</span>
                </div>

                <div className="flex-2 pl-0.5">

                    <div className="flex items-center justify-between gap-2">
                        <div className="flex gap-4 items-center">
                            <span className={`flex text-sm font-bold items-center`}>
                                ${payout && <NumberFlow locales={"en-US"} value={payout.toNumber()} format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }} />}
                            </span>
                            {deltaProp && (
                                <span className={`flex items-center gap-1 px-2 h-6.5 rounded-full text-xs font-bold pb-[1px] w-fit
                                    ${deltaProp.greaterThan(0) ? 'bg-green-400/10 text-success' : 'bg-red-400/10 text-red-400'}`}>
                                    {deltaProp.greaterThan(0) ? '+' : '-'}
                                    {<AnimatedOdds prob={deltaProp.abs()} format="percent" />}
                                </span>
                            )}
                        </div>

                        <div className="w-23">
                            <Button isLoading={isPending} disabled={isPending || !possible} bg={cashoutClicked ? "bg-primary-blue" : "bg-transparent"} width="full" className="border-primary-blue border" height="small" onClick={() => cashoutClicked ? handleCashout() : setCashoutClicked(true)}>
                                <span className={`text-sm font-bold ${cashoutClicked ? 'text-gray-200' : 'text-primary-blue'}`}>
                                    Cashout
                                </span>
                            </Button>
                        </div>



                    </div>

                </div>
            </div>

            {/* MOBILE */}


            <div className="flex md:hidden flex-col gap-5 bg-gray-800/50 p-4 rounded-lg">
                <div className="flex gap-2 items-center">
                    {!market?.isBinary && (
                        <>
                            <OutcomeBadge className={`${bet.side === "y" ? "bg-[#285cac]" : "bg-[#9a45fe]"}`}>{bet.side === "y" ? "Yes" : "No"}</OutcomeBadge>
                            <span className="text-sm font-bold text-white">{bet.outcomeName}</span>
                        </>
                    )}

                    {market?.isBinary && (
                        <OutcomeBadge smaller className={`${market?.outcomes[0].id === bet.outcomeId ? "bg-[#285cac]" : "bg-[#9a45fe]"}`}>{bet.outcomeName}</OutcomeBadge>
                    )}
                </div>

                <div className="grid grid-cols-3 gap-3 w-full">
                    <div className="flex flex-col text-xs gap-2 min-w-0">
                        <span className="text-gray-400 font-bold">STAKE</span>
                        <span className="text-xs font-bold text-white truncate">${bet.pricePaid.toFixed(2)}</span>
                    </div>

                    <div className="flex flex-col text-xs gap-2 min-w-0">
                        <span className="text-gray-400 font-bold">TO WIN</span>
                        <span className="text-xs font-bold text-white truncate">${bet.payout.toFixed(2)}</span>
                    </div>

                    <div className="flex flex-col text-xs gap-2 min-w-0">
                        <span className="text-gray-400 font-bold">CASHOUT FOR</span>
                        <div className="flex items-center gap-2 w-full flex-wrap justify-between">
                            <span className="text-xs font-bold text-white flex-shrink">
                                ${payout && <NumberFlow locales={"en-US"} value={payout.toNumber()} format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }} />}
                            </span>
                        </div>
                    </div>
                </div>

                <div className="w-full">
                    <Button isLoading={isPending} disabled={isPending || !possible} bg={cashoutClicked ? "bg-primary-blue" : "bg-transparent"} width="full" className="border-primary-blue border h-8 flex gap-3" height="small" onClick={() => cashoutClicked ? handleCashout() : setCashoutClicked(true)}>
                        <span className={`text-sm font-bold ${cashoutClicked ? 'text-gray-200' : 'text-primary-blue'}`}>
                            Cashout
                        </span>
                        {/* {deltaProp && (
                            <span className={`flex items-center gap-1 px-2 h-6 rounded-full text-[10px] font-bold pb-[1px] flex-shrink-0
                        ${deltaProp.greaterThan(0) ? 'bg-green-400/10 text-success' : 'bg-red-400/10 text-red-400'}`}>
                                {deltaProp.greaterThan(0) ? '+' : '-'}
                                {<AnimatedOdds prob={deltaProp.abs()} format="percent" />}
                            </span>
                        )} */}
                    </Button>
                </div>
            </div>

        </>
    );
}
