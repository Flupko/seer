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
import { ChevronDown } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import Button from "../Button";
import { CheckNo } from "../Checks";
import { PriceChartDrawer } from "../markets/home/PriceChartDrawer";
import { AnimatedOdds } from "../odds/AnimatedOdds";
import { ArrowOdds } from "../odds/ArrowOdds";
import PriceInput from "../PriceInput";

export default function BetModal({ marketId, initialOutcomeId, initialSide }: { marketId: UUID, initialOutcomeId: number, initialSide: BetSide }) {

    const idempotencyKey = useRef(Math.random().toString()); // generate a unique key per bet attempt

    const [showSuccess, setShowSuccess] = useState(false);
    const betModalContainerRef = useRef<HTMLDivElement>(null);
    const [betModalContainerHeight, setBetModalContainerHeight] = useState(0);

    const closeModal = useModalStore((state) => state.closeModal);

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
            setBetModalContainerHeight(betModalContainerRef.current?.offsetHeight || 0);
            setShowSuccess(true);
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
        <>
            {!showSuccess && <div className={`flex flex-col gap-10 p-6 w-full h-fit`} ref={betModalContainerRef}>

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
                                                    <span className="font-bold text-yes-neon">Bet Yes</span>
                                                    : <span className="font-bold text-no-neon">Bet No</span>}
                                                <div className="w-[3px] h-[3px] rounded-full bg-gray-600 mx-1.5 mt-0.5"></div>
                                                <span className="font-bold">{outcome.name}</span>
                                            </div>
                                        )}

                                        {market.isBinary && (
                                            <span className={`font-bold ${isFirstOutcome ? "text-yes-neon" : "text-no-neon"}`}>Bet {outcome.name}</span>
                                        )}

                                    </div>

                                </div>

                                <div className="flex gap-3 mt-9">

                                    <button className={`flex w-full justify-center gap-3 items-center  ${(market.isBinary ? outcomeId === market.outcomes[0].id : selectedSide == "y") ? "bg-yes-neon text-white hover:brightness-110" : "bg-yes text-yes-text hover:bg-yes-neon hover:text-white brightness-50 hover:brightness-100"}  rounded-md h-12 cursor-pointer active:scale-95 transition-all duration-100`}
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
                                    <button className={`flex w-full justify-center gap-3 items-center ${(market.isBinary ? outcomeId === market.outcomes[1].id : selectedSide == "n") ? "bg-no-neon text-white hover:brightness-110" : "bg-no text-no-text hover:bg-no-neon hover:text-white brightness-50 hover:brightness-100"} rounded-md h-12 cursor-pointer active:scale-95 transition-all duration-100`}
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

            </div >}
            {showSuccess && <div className={`flex flex-col justify-center p-6 w-full`} style={{ height: betModalContainerHeight }}>
                {/* Replace 56px with your header height */}
                <div className="flex flex-col gap-4 items-center">
                    {/* <Check className="w-32 h-32 filter-(--primary-blue-filter)" /> */}
                    <AnimatedCheckCircle />
                    <p className="text-gray-300 text-sm font-medium py-3 text-center">Your bet has been successfully placed.</p>
                    <div className="mt-5 w-full">

                        <Link className="contents" href="/mybets">
                            <Button bg="bg-neon-blue" width="full" onClick={closeModal}>
                                View Bets
                            </Button>
                        </Link>
                    </div>
                </div>
            </div>}

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

            <CheckNo size="w-2.5 h-2.5" className="text-error bg-white" />
            {children}
        </motion.div>
    )
}


function AnimatedCheckCircle() {
    const size = 100;
    const strokeWidth = 4;          // line thickness
    const radius = (size - strokeWidth) / 2;
    const circumference = 2 * Math.PI * radius;

    // circle dasharray: skip part of stroke for top-left gap
    const dashArray = circumference - 14;  // 14 ≈ small visual gap
    const dashOffset = circumference * 0.06; // offset for gap start

    return (
        <motion.svg
            width={size}
            height={size}
            viewBox={`0 0 ${size} ${size}`}
            className="text-primary-blue"
            initial="hidden"
            animate="visible"
            transition={{ staggerChildren: 0.1 }}
        >
            {/* Outer circle with a small gap (top-left) */}
            <motion.circle
                cx={size / 2}
                cy={size / 2}
                r={radius}
                fill="none"
                stroke="currentColor"
                strokeWidth={strokeWidth}
                strokeLinecap="round"
                strokeDasharray={`${dashArray} ${circumference - dashArray}`}
                strokeDashoffset={dashOffset}
                variants={{
                    hidden: { pathLength: 0, opacity: 0 },
                    visible: { pathLength: 1, opacity: 1 },
                }}
                transition={{
                    duration: 0.45,
                    ease: "easeInOut",
                }}
            />

            {/* Checkmark path */}
            <motion.path
                fill="none"
                stroke="currentColor"
                strokeWidth={strokeWidth}
                strokeLinecap="round"
                strokeLinejoin="round"
                d={`M${size * 0.32} ${size * 0.53} L${size * 0.45} ${size * 0.65} 
            L${size * 0.68} ${size * 0.38}`}
                variants={{
                    hidden: { pathLength: 0, opacity: 0 },
                    visible: { pathLength: 1, opacity: 1 },
                }}
                transition={{
                    duration: 0.3,
                    delay: 0.37,
                    ease: "easeOut",
                }}
            />
        </motion.svg>
    );
}