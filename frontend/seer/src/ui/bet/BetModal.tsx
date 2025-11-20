import { useWebSocket } from "@/app/WsProvider";
import { getMarketById, postBet } from "@/lib/api";
import { BetSide, MarketView } from "@/lib/definitions";
import { useOdds } from "@/lib/hooks/useOdds";
import { possiblePayoutProbForSpend } from "@/lib/lslmsr/lslmsr";
import { useBalanceQuery } from "@/lib/queries/useBalanceQuery";
import { useUserQuery } from "@/lib/queries/useUserQuery";
import { useModalStore } from "@/lib/stores/modal";
import NumberFlow from '@number-flow/react';
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { UUID } from "crypto";
import { Decimal } from "decimal.js";
import { ChevronDown, X } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import Image from "next/image";
import { useEffect, useRef, useState } from "react";
import Button from "../Button";
import { PriceChartDrawer } from "../markets/home/PriceChartDrawer";
import { AnimatedOdds } from "../odds/AnimatedOdds";
import { ArrowOdds } from "../odds/ArrowOdds";
import PriceInput from "../PriceInput";

export default function BetModal({ marketId, initialOutcomeId, initialSide }: { marketId: UUID, initialOutcomeId: number, initialSide: BetSide }) {

    const idempotencyKey = useRef(Math.random().toString()); // generate a unique key per bet attempt

    const priceInputRef = useRef<HTMLInputElement>(null);

    // FOR NON BINARY MARKETS
    const [selectedSide, setSelectedSide] = useState<BetSide>(initialSide);

    // BINARY MARKETS
    const [outcomeId, setOutcomeId] = useState<number>(initialOutcomeId);

    const [priceChartShow, setPriceChartShow] = useState<boolean>(true);

    const queryClient = useQueryClient();

    const { data: market } = useQuery<MarketView>({
        queryKey: ['market', marketId],
        queryFn: () => getMarketById(marketId),
        staleTime: Infinity,
    })

    const { data: balance } = useBalanceQuery("USDT")

    const { mutate, isSuccess, isPending } = useMutation({
        mutationFn: postBet,
        onSuccess: () => {
            openModal("betSuccess", {});
            queryClient.invalidateQueries({ queryKey: ['userBets'] });
        }
    });

    const { data: user } = useUserQuery();
    const { openModal } = useModalStore();



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
    const isFirstOutcome = market?.outcomes[0].id == outcome?.id;

    if (!market || !outcome) return null;



    const [possible, payout, probability] = (amount && !amount.isZero()) ? possiblePayoutProbForSpend(market, outcomeId, amount, selectedSide) : [false, new Decimal(0), new Decimal(0)];

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

        console.log("possible:", possible, "payout:", payout.toString());

        // Unfocus the price input to ensure value is committed
        priceInputRef.current?.blur();

        mutate({
            marketId: market.id,
            side: selectedSide,
            minWantedGain: payout,
            outcomeId: outcome.id,
            betAmount: amount,
            currency: "USDT",
            idempotencyKey: idempotencyKey.current,
        });
    }

    return (
        <div className="flex flex-col gap-10 p-6 w-full h-fit">

            {/* Outcomes select input */}


            <AnimatePresence key="bet-drawer-content">
                <motion.div className="space-y-4" style={{ scrollbarColor: "var(--color-gray-800) transparent", scrollbarWidth: "thin" }}>

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
                        transition={{ duration: 0.25, ease: "easeOut" }}
                        className="space-y-6">


                        <div>
                            <div className="flex items-center gap-4">
                                {market.imgKey && (
                                    <div className="flex-shrink-0 w-14 h-14 rounded-lg overflow-hidden">
                                        <Image
                                            src={market.imgKey}
                                            alt={market.name}
                                            width={250}
                                            height={250}
                                            className="object-cover w-full h-full"
                                        />
                                    </div>)}
                                <div className="flex flex-col gap-1.5 text-sm">
                                    <h2 className="text-sm font-medium h-fit leading-5.5 line-clamp-2">{market.name}</h2>
                                    {!market.isBinary && (
                                        <div className="flex items-center">
                                            {selectedSide === "y" ?
                                                <span className="font-bold text-[#285cac]">Bet Yes</span>
                                                : <span className="font-bold text-[#9a45fe]">Bet No</span>}
                                            <div className="w-[3px] h-[3px] rounded-full bg-gray-600 mx-1.5 mt-0.5"></div>
                                            <span className="font-bold">{outcome.name}</span>
                                        </div>
                                    )}

                                    {market.isBinary && (
                                        <span className={`font-bold ${isFirstOutcome ? "text-[#285cac]" : "text-[#9a45fe]"}`}>Bet {outcome.name}</span>
                                    )}

                                </div>

                            </div>

                            <div className="flex gap-3 mt-9">

                                <button className={`flex w-full justify-center gap-3 items-center bg-[#285cac]/90 ${(market.isBinary ? outcomeId === market.outcomes[0].id : selectedSide == "y") ? "hover:brightness-110" : "brightness-40 hover:brightness-50"}  rounded-md h-12 cursor-pointer active:scale-95 transition-all duration-100`}
                                    onClick={() => market.isBinary ? setOutcomeId(market.outcomes[0].id) : setSelectedSide("y")}>
                                    <span className="flex items-baseline gap-2">
                                        <span className="font-medium text-[15px]">{market.isBinary ? market.outcomes[0].name : "Yes"}</span>
                                        <span className="font-bold text-lg"><ArrowOdds
                                            prob={market.isBinary ? market.outcomes[0].priceYes : outcome.priceYes}
                                            format={oddsFormat}
                                        />
                                        </span>
                                    </span>

                                </button>
                                <button className={`flex w-full justify-center gap-3 items-center bg-[#9a45fe]/90 ${(market.isBinary ? outcomeId === market.outcomes[1].id : selectedSide == "n") ? "hover:brightness-110" : "brightness-40 hover:brightness-50"} rounded-md h-12 cursor-pointer active:scale-95 transition-all duration-100`}
                                    onClick={() => market.isBinary ? setOutcomeId(market.outcomes[1].id) : setSelectedSide("n")}>
                                    <span className="flex items-baseline gap-2">
                                        <span className="font-bold text-[15px]">{market.isBinary ? market.outcomes[1].name : "No"}</span>
                                        <span className="font-bold text-lg"><ArrowOdds
                                            prob={market.isBinary ? market.outcomes[1].priceYes : outcome.priceNo}
                                            format={oddsFormat}
                                        />
                                        </span>
                                    </span>

                                </button>
                            </div>





                            <div className="flex flex-col gap-4.5 mt-8">
                                <div className="text-sm flex items-center justify-between cursor-pointer select-none" onClick={() => setPriceChartShow(!priceChartShow)}>
                                    <span className="text-gray-400">Chance - 24h</span>
                                    <motion.span
                                        animate={{ rotate: priceChartShow ? 180 : 0 }}
                                        transition={{ duration: 0.13, ease: "linear" }}
                                    >
                                        <ChevronDown size={20} strokeWidth={1.2} className="text-gray-400" />
                                    </motion.span>
                                </div>
                                {priceChartShow &&
                                    <PriceChartDrawer data={outcome.priceCharts?.find(chart => chart.timeframe === "24h")?.prices || []} side={selectedSide} />}
                            </div>





                            <div className="flex flex-col gap-2 mt-6">
                                <label className="text-sm text-gray-400">Amount</label>
                                <PriceInput placeholder="Enter stake" onValueChange={(v) => {
                                    setAmount(v ? new Decimal(v) : undefined);
                                }} ref={priceInputRef} />
                            </div>

                            {/* Payout display */}
                            <div className="flex flex-col gap-1.5 mt-6">
                                <div className="text-sm  flex justify-between">
                                    <span className="text-gray-400">To Win :</span> {

                                        <NumberFlow
                                            value={payout.toNumber()}
                                            locales="en-US"
                                            format={{ style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                                            className="text-green-400 font-bold"
                                        />
                                    }

                                </div>
                                <div className="text-sm  flex justify-between">
                                    <span className="text-gray-400">Best Odd :</span>
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









                        <Button className="w-full" disabled={!!user && (!possible || hasInsufficientFunds)} isLoading={isPending} bg="bg-primary-blue" width="full"
                            onClick={handleClickBet}
                        >
                            <span className="font-medium">{user ? "Place Bet" : "Register to Bet"}</span>
                        </Button>

                    </motion.div>


                </motion.div >
            </AnimatePresence>

        </div >

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