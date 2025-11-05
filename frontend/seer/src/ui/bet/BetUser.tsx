import { cashoutBet, getMarket } from "@/lib/api";
import { Bet } from "@/lib/definitions";
import { possiblePayoutDeltaForCashout } from "@/lib/lslmsr/lslmsr";
import { formatOdds } from "@/lib/odds";
import { usePrefs } from "@/lib/stores/prefs";
import NumberFlow from "@number-flow/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Image from "next/image";
import { useRef, useState } from "react";
import DollarIcon from "../../../public/icons/dollar.svg";
import Button from "../Button";
import { AnimatedOdds } from "../odds/AnimatedOdds";
import { toastStyled } from "../Toast";

export default function BetUser({ bet }: { bet: Bet }) {

    const idempotencyKey = useRef(Math.random().toString());

    const { data: market } = useQuery({
        queryKey: ['market', bet.marketId],
        queryFn: () => getMarket(bet.marketId),
        staleTime: Infinity,
    })

    const queryClient = useQueryClient();

    const { mutate, isSuccess, isPending } = useMutation({
        mutationFn: cashoutBet,
        onSuccess: () => {
            // Invalidate 
            toastStyled("Bet cashed out successfully", { type: "success", autoClose: 1500 });
            queryClient.invalidateQueries({ queryKey: ['userBets'] });
        },
    });

    const [cashoutClicked, setCashoutClicked] = useState(false);
    const isResolved = bet.status === 'won' || bet.status === 'lost' || bet.status === 'cashedOut';


    const oddsFormat = usePrefs((state) => state.oddsFormat);
    const [possible, deltaProp, payout] = market ? possiblePayoutDeltaForCashout(market, bet) : [null, null, null];
    console.log("delta prop", possible && deltaProp?.toNumber());

    const handleCashout = () => {

        if (!cashoutClicked || !possible || isResolved) return;

        mutate({
            betId: bet.id,
            minWantedGain: payout,
            idempotencyKey: idempotencyKey.current,
        });
    }


    return (
        <div>

            <div className="bg-gray-700 p-4 px-4.5 flex flex-col gap-8 rounded-t-lg relative">


                <div className="flex justify-between items-center">
                    <div className="flex items-center gap-3">
                        {bet.marketImgKey && (
                            <div className="flex-shrink-0 w-9 h-9 rounded-md overflow-hidden bg-gray-900">
                                <Image
                                    src={bet.marketImgKey}
                                    alt={bet.marketName}
                                    width={24}
                                    height={24}
                                    className="object-cover w-full h-full"
                                />
                            </div>)}
                        <h2 className="text-sm font-bold h-fit leading-5.5 line-clamp-1">{bet.marketName}</h2>


                    </div>

                    {isResolved && (
                        <span className={`text-xs font-bold py-[0.15rem] px-[0.3rem] rounded-md text-grayscale-black whitespace-nowrap
                         ${bet.status === "won" && "bg-success"}
                         ${bet.status === "lost" && "bg-error"}
                          ${bet.status === "cashedOut" && "bg-yellow-500"}`}

                        >
                            {bet.status === "won" && "WIN"}
                            {bet.status === "lost" && "LOSS"}
                            {bet.status === "cashedOut" && "CASH OUT"
                            }
                        </span>)
                    }
                </div>



                <span className="absolute bg-gray-900 rounded-full w-4 h-4 bottom-0 translate-y-1/2 -left-0.5 -translate-x-1/2"></span>
                <span className="absolute bg-gray-900 rounded-full w-4 h-4 bottom-0 translate-y-1/2 -right-0.5 translate-x-1/2"></span>

                <div className="absolute bottom-0 left-0 w-full translate-y-1/2">
                    <div className="border-t-4 border-dotted  border-gray-900" />
                </div>

            </div>



            <div className="bg-gray-800 p-4.5 pt-3.5 rounded-b-lg flex flex-col gap-5.5">


                <div className="flex flex-col gap-5">
                    <div className="flex items-center justify-between">
                        <span className="text-sm font-bold text-primary-blue">
                            {bet.outcomeName}
                        </span>

                        <span className="text-sm font-bold text-white">
                            {formatOdds(bet.avgPrice, oddsFormat)}
                        </span>
                    </div>


                    {/* Payout display */}
                    <div className="flex flex-col gap-1.5">
                        <div className="text-sm text-gray-300 flex justify-between">
                            <span className="text-gray-300">
                                Your Stake :
                            </span>
                            <span className="text-white font-bold">
                                ${bet.pricePaid.toFixed(2)}
                            </span>

                        </div>
                        <div className="text-sm flex justify-between">
                            <span className="text-gray-300">
                                {bet.status === "active" && "To Win"}
                                {bet.status === "won" && "You Won"}
                                {bet.status === "lost" && "No Payout"}
                                {bet.status === "cashedOut" && "Cashed"}
                            </span> {
                                <span className={`${bet.status === "lost" ? "text-gray-400" : "text-green-400"} font-bold`}>
                                    ${bet.status === "lost" ? "0.00" : bet.status === "cashedOut" && bet.cashedOut ? bet.cashedOut?.toFixed(2) : bet.payout.toFixed(2)}
                                </span>
                            }
                        </div>

                    </div>
                </div>




                {/* Sell button */}
                {/* Cashout gain and percentage display */}

                {!isResolved &&
                    <div className="mt">
                        <Button isLoading={isPending} disabled={isPending || !possible} bg={cashoutClicked ? "bg-success" : "bg-transparent"} width="full" className="border-success border" height="extraSmall" onClick={() => cashoutClicked ? handleCashout() : setCashoutClicked(true)}>
                            <div className="flex justify-center w-full items-center px-4 gap-4">

                                <div className="flex items-center gap-2">
                                    {/* Main cashout amount */}
                                    <span className={`text-sm font-bold ${cashoutClicked ? 'text-grayscale-black' : 'text-success'}`}>
                                        {cashoutClicked ? 'Confirm Cashout ' : 'Cashout for '}
                                    </span>

                                    <span className={`flex text-sm font-bold items-center gap-1.5 ${cashoutClicked ? 'text-grayscale-black' : 'text-success'}`}>
                                        <DollarIcon className="w-4 h-4" />
                                        {possible && <NumberFlow locales={"en-US"} value={payout.toNumber()} format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }} />}
                                    </span>
                                </div>



                                {/* Delta percentage badge */}
                                {deltaProp && !cashoutClicked && (
                                    <span className={`flex items-center gap-1 px-2 h-5 rounded-full text-xs font-bold pb-[1px]

                                    ${deltaProp.greaterThan(0) ? 'bg-green-400/10 text-success' : 'bg-red-400/10 text-red-400'}`}>
                                        {deltaProp.greaterThan(0) ? '+' : '-'}
                                        {<AnimatedOdds prob={deltaProp.abs()} format="percent" />}
                                    </span>
                                )}
                            </div>
                        </Button>
                    </div>

                }


            </div>
        </div >
    );
}