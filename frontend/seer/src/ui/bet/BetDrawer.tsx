import { useWebSocket } from "@/app/WsProvider";
import { getMarket, postBet } from "@/lib/api";
import { MarketView } from "@/lib/definitions";
import { useOdds } from "@/lib/hooks/useOdds";
import { possiblePayoutProbForSpend } from "@/lib/lslmsr/lslmsr";
import { useBalanceQuery } from "@/lib/queries/useBalanceQuery";
import { useUserQuery } from "@/lib/queries/useUserQuery";
import { useDrawerStore } from "@/lib/stores/drawer";
import { useModalStore } from "@/lib/stores/modal";
import NumberFlow from '@number-flow/react';
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { UUID } from "crypto";
import { Decimal } from "decimal.js";
import { X } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import Image from "next/image";
import { useEffect, useRef, useState } from "react";
import Button from "../Button";
import DrawerHeader from "../drawer/DrawerHeader";
import { PriceChart } from "../markets/PriceChart";
import MenuVertical from "../menu_small_vertical/MenuVertical";
import { AnimatedOdds } from "../odds/AnimatedOdds";
import PriceInput from "../PriceInput";

export default function BetDrawer({ marketId, initialOutcomeId }: { marketId: UUID, initialOutcomeId: number }) {

    const idempotencyKey = useRef(Math.random().toString()); // generate a unique key per bet attempt

    const openDrawer = useDrawerStore((state) => state.openDrawer);
    const priceInputRef = useRef<HTMLInputElement>(null);

    const queryClient = useQueryClient();

    const { data: market } = useQuery<MarketView>({
        queryKey: ['market', marketId],
        queryFn: () => getMarket(marketId),
        staleTime: Infinity,
    })

    const { data: balance } = useBalanceQuery("USDT")

    const { mutate, isSuccess, isPending } = useMutation({
        mutationFn: postBet,
        onSuccess: () => {
            openDrawer("betSuccess", {});
            queryClient.invalidateQueries({ queryKey: ['userBets'] });
        }
    });

    const { data: user } = useUserQuery();
    const { openModal } = useModalStore();

    const [outcomeId, setOutcomeId] = useState<number>(initialOutcomeId);

    const ws = useWebSocket();
    // On mount add the listener
    useEffect(() => {
        if (!ws) return;

        ws.emit({ type: "get:market_state", payload: { marketId } });

        const off = ws.onConnect(() => {
            ws.emit({ type: "get:market_state", payload: { marketId } });
        });

        return () => {
            off();
        }

    }, [ws, marketId]);

    useEffect(() => {
        setOutcomeId(initialOutcomeId);
    }, [initialOutcomeId]);


    const [amount, setAmount] = useState<Decimal | undefined>(undefined);
    const { oddsFormat } = useOdds();

    const outcome = market?.outcomes.find(o => o.id === outcomeId);
    if (!market || !outcome) return null;



    const [possible, payout, probability] = (amount && !amount.isZero()) ? possiblePayoutProbForSpend(market, outcomeId, amount) : [false, new Decimal(0), new Decimal(0)];

    const hasInsufficientFunds = (balance && amount) ? balance.balance.lessThan(amount) : false;
    const betTooMuch = (amount && !amount.isZero()) && !possible
    const isErrored = betTooMuch || hasInsufficientFunds;

    const handleClickBet = () => {
        if (!user) {
            openModal("auth", { selectedTab: 'register' });
            return;
        }

        if (!market || !balance || !amount) return;
        if (!possible || hasInsufficientFunds) return;

        // Unfocus the price input to ensure value is committed
        priceInputRef.current?.blur();

        mutate({
            marketId: market.id,
            minWantedGain: payout,
            outcomeId: outcome.id,
            betAmount: amount,
            currency: "USDT",
            idempotencyKey: idempotencyKey.current,
        });
    }

    return (
        <>
            <DrawerHeader title="Place Bet" />

            {/* Outcomes select input */}


            <AnimatePresence key="bet-drawer-content">
                <motion.div layout className="px-5 pt-4 space-y-4 h-[calc(100vh-76px)] overflow-y-auto" style={{ scrollbarColor: "var(--color-gray-800) transparent", scrollbarWidth: "thin" }}>



                    {isErrored &&
                        <div className="flex flex-col gap-4">
                            {
                                hasInsufficientFunds &&
                                <ErrorDrawer>
                                    Insufficient funds
                                </ErrorDrawer>
                            }

                            {
                                betTooMuch &&
                                <ErrorDrawer>
                                    Current bet amount too high for selected outcome
                                </ErrorDrawer>
                            }
                        </div>
                    }





                    <motion.div
                        layout
                        transition={{ duration: 0.25, ease: "easeOut" }}
                        className="space-y-5">


                        <div>

                            <div className="bg-gray-700 p-4.5 pb-7 flex flex-col gap-8 rounded-t-lg relative border border-transparent border-b-0">
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

                                <span className="absolute bg-gray-900 rounded-full w-4 h-4 bottom-0 translate-y-1/2 -left-0.5 -translate-x-1/2"></span>
                                <span className="absolute bg-gray-900 rounded-full w-4 h-4 bottom-0 translate-y-1/2 -right-0.5 translate-x-1/2"></span>

                                <div className="absolute bottom-0 left-0 w-full translate-y-1/2">
                                    <div className="border-t-4 border-dotted  border-gray-900" />
                                </div>

                            </div>



                            <div className="bg-gray-800 p-4.5 pt-7 rounded-b-lg border border-t-0 border-transparent flex flex-col">

                                <div className="flex items-center justify-between">
                                    <div className="w-50">
                                        <MenuVertical
                                            choices={market.outcomes.map(o => ({ element: o.name, value: o.id }))}
                                            value={outcomeId}
                                            onChange={(value) => setOutcomeId(value)}
                                            height="h-11"
                                            bg="bg-gray-900"
                                        />
                                    </div>

                                    <span className="text-sm font-bold text-primary-blue">
                                        {<AnimatedOdds
                                            prob={outcome.price}
                                            format={oddsFormat}
                                        />
                                        }
                                    </span>
                                </div>

                                <PriceChart data={outcome.priceCharts?.find(chart => chart.timeframe === "24h")?.prices || []} />

                                <div className="flex flex-col gap-1.5 mb-6.5">
                                    <label className="text-sm font-medium text-gray-300">Amount</label>
                                    <PriceInput placeholder="Enter stake" onValueChange={(v) => {
                                        setAmount(v ? new Decimal(v) : undefined);
                                    }} ref={priceInputRef} />
                                </div>

                                {/* Payout display */}
                                <div className="flex flex-col gap-1.5">
                                    <div className="text-sm  flex justify-between">
                                        <span className="text-gray-300">To Win :</span> {

                                            <NumberFlow
                                                value={payout.toNumber()}
                                                locales="en-US"
                                                format={{ style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                                                className="text-green-400 font-bold"
                                            />
                                        }

                                    </div>
                                    <div className="text-sm  flex justify-between">
                                        <span className="text-gray-300">Best Odd :</span>
                                        {
                                            <span className="text-green-400 font-bold">
                                                {<AnimatedOdds
                                                    prob={probability}
                                                    format={oddsFormat}
                                                />
                                                }
                                            </span>
                                        }
                                    </div>
                                </div>
                            </div>
                        </div>






                        <Button className="w-full" disabled={!!user && (!possible || hasInsufficientFunds)} isLoading={isPending} bg="bg-neon-blue" width="full"
                            onClick={handleClickBet}
                        >
                            <span className="font-medium">{user ? "Place Bet" : "Register to Bet"}</span>
                        </Button>

                    </motion.div>


                </motion.div >
            </AnimatePresence>

        </>

    )




}

function ErrorDrawer({ children }: { children: React.ReactNode }) {
    return (
        <motion.div className="text-sm font-bold bg-error py-3.5 px-5 rounded-lg flex items-center gap-3"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2, ease: "easeIn" }} >

            <span className="bg-white rounded-full p-[0.2rem]">
                <X className="w-2.5 h-2.5 text-error" strokeWidth={5} />
            </span>
            {children}
        </motion.div>
    )
}