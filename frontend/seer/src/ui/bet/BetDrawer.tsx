import { useWebSocket } from "@/app/WsProvider";
import { getMarket } from "@/lib/api";
import { MarketView } from "@/lib/definitions";
import { useOdds } from "@/lib/hooks/useOdds";
import { possiblePayoutPropForSpend } from "@/lib/lslmsr/lslmsr";
import { useUserQuery } from "@/lib/queries/useUserQuery";
import NumberFlow from '@number-flow/react';
import { useQuery } from "@tanstack/react-query";
import { UUID } from "crypto";
import { Decimal } from "decimal.js";
import Image from "next/image";
import { useEffect, useState } from "react";
import { AnimatedOdds } from "../AnimatedOdds";
import Button from "../Button";
import DrawerHeader from "../drawer/DrawerHeader";
import PriceInput from "../PriceInput";

export default function BetDrawer({ marketId, outcomeId }: { marketId: UUID, outcomeId: number }) {

    const { data: market } = useQuery<MarketView>({
        queryKey: ['market', marketId],
        queryFn: () => getMarket(marketId),
        staleTime: Infinity,
    })

    const { data: user } = useUserQuery();

    const [amount, setAmount] = useState<Decimal | undefined>(undefined);

    const { oddsFormat } = useOdds();
    const ws = useWebSocket();

    useEffect(() => {

    }, [marketId, ws]);

    const outcome = market?.outcomes.find(o => o.id === outcomeId);
    if (!market || !outcome) return null;

    const [possible, payout, probability] = (amount && !amount.isZero()) ? possiblePayoutPropForSpend(market, outcomeId, amount) : [false, new Decimal(0), new Decimal(0)];
    return (
        <>
            <DrawerHeader title="Place Bet" />
            <div className="px-5 pt-4">


                <div className="bg-gray-700 p-4.5 pb-4 flex flex-col gap-6 rounded-t-lg relative border border-gray-600">
                    <div className="flex items-center gap-4">
                        {market.imgKey && (
                            <div className="flex-shrink-0 w-12 h-12 rounded-md overflow-hidden bg-gray-900">
                                <Image
                                    src={market.imgKey}
                                    alt={market.name}
                                    width={48}
                                    height={48}
                                    className="object-cover w-full h-full"
                                />
                            </div>)}
                        <h2 className="text-md font-bold h-fit leading-5.5 line-clamp-2">{market.name}</h2>
                    </div>

                    <div className="flex items-center justify-between">
                        <span className="text-sm font-bold text-primary-blue">{outcome.name}</span>
                        <span className="text-sm font-bold text-white">
                            {<AnimatedOdds
                                prob={outcome.price}
                                format={oddsFormat}
                            />
                            }
                        </span>
                    </div>

                    <span className="absolute bg-gray-900 rounded-full w-3.5 h-3.5 bottom-0 translate-y-1/2 left-0 -translate-x-1/2"></span>
                    <span className="absolute bg-gray-900 rounded-full w-3.5 h-3.5 bottom-0 translate-y-1/2 right-0 translate-x-1/2"></span>
                </div>

                <div className="bg-gray-800 p-4.5 pt-4 rounded-b-lg border border-t-0 border-gray-600">
                    <div className="">
                        <label className="text-sm font-medium text-gray-400 mb-2 block">Amount</label>
                        <PriceInput placeholder="Enter stake" onValueChange={(v) => {
                            setAmount(v ? new Decimal(v) : undefined);
                        }} />
                    </div>

                    {/* Payout display */}
                    <div className="flex flex-col gap-1 mt-5">
                        <span className="text-sm text-gray-300 flex justify-between">
                            <span>Payout :</span> {

                                <NumberFlow
                                    value={payout.toNumber()}
                                    locales="en-US"
                                    format={{ style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                                    className="text-green-400 font-bold"
                                />
                            }

                        </span>
                        <span className="text-sm text-gray-300 flex justify-between">
                            <span>Best Odd :</span> {
                                <span className="text-green-400 font-bold">
                                    {<AnimatedOdds
                                        prob={probability}
                                        format={oddsFormat}
                                    />
                                    }
                                </span>
                            }
                        </span>
                    </div>
                </div>


                <Button className="w-full mt-6" disabled={!possible || !amount || amount.isZero()} bg="bg-neon-blue" width="full">
                    <span className="font-bold">Place Bet</span>
                </Button>






            </div >



        </>

    )




}