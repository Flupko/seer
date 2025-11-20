import { cashoutBet, getMarketById } from "@/lib/api";
import { Bet } from "@/lib/definitions";
import { possiblePayoutDeltaForCashout } from "@/lib/lslmsr/lslmsr";
import NumberFlow from "@number-flow/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Image from "next/image";
import { useRef, useState } from "react";
import Button from "../Button";
import { OutcomeBadge } from "../markets/OutcomeBadge";
import { AnimatedOdds } from "../odds/AnimatedOdds";
import { toastStyled } from "../Toast";

export default function BetUser({ bet }: { bet: Bet }) {

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


    const isResolved = bet.status === 'won' || bet.status === 'lost' || bet.status === 'cashedOut';

    const handleCashout = () => {

        if (!cashoutClicked || !possible) return;

        mutate({
            betId: bet.id,
            minWantedGain: payout,
            idempotencyKey: idempotencyKey.current,
        });
    }

    if (!market) return null;

    // bet.status = "won"

    const [possible, deltaProp, payout] = market ? possiblePayoutDeltaForCashout(market, bet) : [null, null, null];

    return (
        <>
            <div className="hidden lg:flex items-center justify-between bg-gray-800/50 p-3 rounded-lg">
                {/* <div className="flex-2 pl-0.5">
                    <span className="text-sm font-bold text-white">{bet.outcomeName}</span>
                    <div className={`text-sm font-bold flex items-center justify-center text-white rounded-sm h-7 w-10 ${bet.side === "y" ? "bg-[#285cac]" : "bg-[#9a45fe]"}`}>{bet.side === "y" ? "Yes" : "No"}</div>
                </div> */}

                <div className="flex-2 flex items-center gap-4">
                    {market.imgKey && (
                        <div className="flex-shrink-0 w-12 h-12 rounded-md overflow-hidden">
                            <Image
                                src={market.imgKey}
                                alt={market.name}
                                width={250}
                                height={250}
                                className="object-cover w-full h-full"
                            />
                        </div>)}
                    <div className="flex flex-col gap-1.5 text-sm">
                        <h2 className="text-sm font-medium h-fit leading-5.5 line-clamp-1 text-ellipsis break-all">
                            {market?.name}
                        </h2>
                        <div className="flex items-center">

                            {!market?.isBinary && (<>
                                <span className="font-bold text-sm">{bet.outcomeName}</span>
                                <div className="w-[3px] h-[3px] rounded-full bg-gray-600 mx-1.5 mt-0.5"></div>
                                <OutcomeBadge smaller className={`text-xs ${bet.side === "y" ? "bg-[#285cac]" : "bg-[#9a45fe]"}`}>{bet.side === "y" ? "Yes" : "No"}</OutcomeBadge></>
                            )}
                            {market?.isBinary && (
                                <OutcomeBadge smaller className={`${market.outcomes[0].id === bet.outcomeId ? "bg-[#285cac]" : "bg-[#9a45fe]"}`}>{bet.outcomeName}</OutcomeBadge>
                            )}
                        </div>
                    </div>

                </div>

                <div className="flex-1 pl-0.5">
                    <span className="text-sm font-bold text-white">${bet.pricePaid.toFixed(2)}</span>
                </div>
                <div className="flex-1 pl-0.5">
                    <span className="text-sm font-bold text-white">${bet.payout.toFixed(2)}</span>
                </div>



                <div className={`flex items-center gap-2 ${isResolved ? "w-50" : "flex-2 justify-between"}`}>

                    {!isResolved && (
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
                    )

                    }


                    {!isResolved && (
                        <div className="w-25 justify-self-end">
                            <Button isLoading={isPending} disabled={isPending || !possible} bg={cashoutClicked ? "bg-primary-blue" : "bg-transparent"} width="full" className="border-primary-blue border h-8" height="small" onClick={() => cashoutClicked ? handleCashout() : setCashoutClicked(true)}>
                                <span className={`text-sm font-bold ${cashoutClicked ? 'text-gray-200' : 'text-primary-blue'}`}>
                                    Cashout
                                </span>
                            </Button>
                        </div>)}

                    {isResolved && (
                        <div className="flex gap-2 items-center">

                            <span className={`text-xs font-bold py-1 px-1.5 rounded-sm text-white whitespace-nowrap
                                    ${bet.status === "won" && "bg-green-600"}
                                    ${bet.status === "lost" && "bg-orange-700"}
                                    ${bet.status === "cashedOut" && "bg-indigo-600"}`}

                            >
                                {bet.status === "won" && "WIN"}
                                {bet.status === "lost" && "LOSS"}
                                {bet.status === "cashedOut" && "CASHED"
                                }
                            </span>


                            {bet.status === "cashedOut" && (<div className="text-sm flex gap-2 items-center">
                                <span className="text-gray-500 font-bold tracking-wider text-xs">FOR </span>
                                <span className="text-white font-bold"> ${bet.cashedOut?.toFixed(2)} </span>

                            </div>)}


                        </div>)
                    }
                </div>

            </div>



            {/* MOBILE */}
            <div className="flex lg:hidden flex-col gap-5 bg-gray-800/50 p-4 rounded-lg">
                <div className="flex justify-between items-center min-w-0">
                    <div className="flex items-center gap-4 min-w-0">
                        {market.imgKey && (
                            <div className="flex-shrink-0 w-12 h-12 rounded-md overflow-hidden">
                                <Image
                                    src={market.imgKey}
                                    alt={market.name}
                                    width={250}
                                    height={250}
                                    className="object-cover w-full h-full"
                                />
                            </div>)}
                        <div className="flex flex-col gap-1.5 text-sm min-w-0">
                            <h2 className="text-sm font-medium leading-5.5 line-clamp-1 text-ellipsis break-all">
                                {market.name}
                            </h2>
                            <div className="flex items-center min-w-0">
                                {!market?.isBinary && (<>
                                    <span className="font-bold min-w-0 line-clamp-2 text-ellipsis text-sm">
                                        {bet.outcomeName}
                                    </span>
                                    <div className="w-[3px] h-[3px] rounded-full bg-gray-600 mx-1.5 mt-0.5"></div>
                                    <OutcomeBadge smaller className={`${bet.side === "y" ? "bg-[#285cac]" : "bg-[#9a45fe]"}`}>{bet.side === "y" ? "Yes" : "No"}</OutcomeBadge></>
                                )}

                                {market?.isBinary && (
                                    <OutcomeBadge smaller className={`${market.outcomes[0].id === bet.outcomeId ? "bg-[#285cac]" : "bg-[#9a45fe]"}`}>{bet.outcomeName}</OutcomeBadge>
                                )}

                            </div>
                        </div>

                    </div>

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

                    <div className={`flex flex-col text-xs gap-2 min-w-0 ${(isResolved && bet.status !== "cashedOut") && "justify-center"}`}>
                        {!isResolved && (
                            <>
                                <span className="text-gray-400 font-bold">CASHOUT FOR</span>
                                <div className="flex items-center gap-2 w-full flex-wrap justify-between">
                                    <span className="text-xs font-bold text-white flex-shrink">
                                        ${payout && <NumberFlow locales={"en-US"} value={payout.toNumber()} format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }} />}
                                    </span>
                                    {/* {deltaProp && (
                                <span className={`flex items-center gap-1 px-2 h-6.5 rounded-full text-[10px] font-bold pb-[1px] flex-shrink-0
                        ${deltaProp.greaterThan(0) ? 'bg-green-400/10 text-success' : 'bg-red-400/10 text-red-400'}`}>
                                    {deltaProp.greaterThan(0) ? '+' : '-'}
                                    {<AnimatedOdds prob={deltaProp.abs()} format="percent" />}
                                </span>
                            )} */}
                                </div>
                            </>
                        )}
                        {isResolved && (
                            <div className="flex flex-col gap-2">
                                <span className={`text-xs font-bold py-1 px-1.5 rounded-sm text-white whitespace-nowrap w-fit
                                ${bet.status === "won" && "bg-green-600"}
                                ${bet.status === "lost" && "bg-orange-700"}
                                ${bet.status === "cashedOut" && "bg-indigo-600"}`}
                                >
                                    {bet.status === "won" && "WIN"}
                                    {bet.status === "lost" && "LOSS"}
                                    {bet.status === "cashedOut" && "CASHED"
                                    }
                                </span>
                                {bet.status === "cashedOut" && (<div className="flex gap-1.5 items-center">
                                    <span className="text-gray-400 font-bold tracking-wider">FOR </span>
                                    <span className="text-white font-bold"> ${bet.cashedOut?.toFixed(2)} </span>

                                </div>)
                                }
                            </div>
                        )
                        }
                    </div>


                </div>

                {!isResolved && (
                    <div className="w-full">


                        <Button isLoading={isPending} disabled={isPending || !possible} bg={cashoutClicked ? "bg-primary-blue" : "bg-transparent"} width="full" className="border-primary-blue border h-8" height="small" onClick={() => cashoutClicked ? handleCashout() : setCashoutClicked(true)}>
                            <span className={`text-sm font-bold ${cashoutClicked ? 'text-gray-200' : 'text-primary-blue'}`}>
                                Cashout
                            </span>
                        </Button>
                    </div>)}
            </div>



        </>
    );
}
