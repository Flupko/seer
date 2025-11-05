"use client";

import { searchMarket } from "@/lib/api";
import { Category, MarketSearch } from "@/lib/definitions";
import { marketSearchKey } from "@/lib/queries/marketSearchKey";
import PrefetchLink from "@/ui/PrefetchLink";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { motion } from "motion/react";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import CategoryButton from "./CategoryButton";

export default function CategoriesMenu({
    categories,
}: {
    categories: Category[];
}) {
    const [disableButton, setDisableButton] = useState("left");
    const ref = useRef<HTMLDivElement>(null);

    const scrollByAmount = (dir: 1 | -1) => {
        if (!ref.current) return;
        const el = ref.current;
        const { clientWidth, scrollLeft, scrollWidth } = el;

        // Calculate desired scroll amount
        const desiredAmount = Math.round(clientWidth * 0.8) * dir;

        // Calculate maximum scrollable distance
        const maxScrollLeft = scrollWidth - clientWidth;

        // Clamp the scroll amount to not exceed boundaries
        let actualAmount = desiredAmount;

        if (dir > 0) {
            // Scrolling right: don't exceed maxScrollLeft
            const remainingScroll = maxScrollLeft - scrollLeft;
            actualAmount = Math.min(desiredAmount, remainingScroll);
        } else {
            // Scrolling left: don't go below 0
            actualAmount = Math.max(desiredAmount, -scrollLeft);
        }

        el.scrollBy({ left: actualAmount, behavior: "smooth" });
    };

    const handleScroll = () => {
        if (!ref.current) return;
        const { scrollLeft, scrollWidth, clientWidth } = ref.current;

        // Add tolerance for floating-point precision issues (especially on mobile)
        const tolerance = 2;
        // Check if scrolled to the very left
        const atStart = scrollLeft <= tolerance;
        // Check if scrolled to the very right
        const atEnd = scrollLeft + clientWidth >= scrollWidth - tolerance;

        if (atStart) {
            setDisableButton("left");
        } else if (atEnd) {
            setDisableButton("right");
        } else {
            setDisableButton("none");
        }
    };

    // Check scroll position on mount
    useEffect(() => {
        handleScroll();
    }, [categories]);

    const activeCategorySlug = useSearchParams().get("category") ?? categories[0]?.slug;

    return (
        <div className="relative">
            {disableButton !== "left" && (
                <motion.button
                    type="button"
                    aria-label="Scroll left"
                    onClick={() => scrollByAmount(-1)}
                    whileTap={{ scale: 0.95 }}
                    className="absolute left-0 -top-0.5 z-10 bg-primary-blue/20 active:bg-primary-blue/40 
                             cursor-pointer w-8.5 h-14.5 border-none rounded-tr-full rounded-br-full 
                             flex items-center justify-center"
                >
                    <ChevronLeft className="text-white w-6 h-6 cursor-pointer mr-1" strokeWidth={0.9} />
                </motion.button>
            )}

            <div
                ref={ref}
                onScroll={handleScroll}
                className="flex overflow-x-auto hide-scrollbar gap-4.5 px-1"
            >
                {categories.map((category) => {
                    const searchParams = new URLSearchParams();
                    searchParams.set("category", category.slug);

                    const search: MarketSearch = {
                        categorySlug: category.slug,
                        page: 1,
                        pageSize: 6,
                        sort: "trending",
                        status: "active",
                    };

                    return (
                        <PrefetchLink
                            href={`/?${searchParams.toString()}`}
                            key={category.id}
                            className="contents"
                            infiniteQueries={[{
                                queryKey: marketSearchKey(search),
                                queryFn: ({ pageParam = 1 }) =>
                                    searchMarket({ ...search, page: pageParam } as MarketSearch),
                            }]}
                        >
                            <CategoryButton
                                category={category}
                                active={category.slug === activeCategorySlug}
                            />
                        </PrefetchLink>
                    );
                })}
            </div>

            {disableButton !== "right" && (
                <motion.button
                    type="button"
                    aria-label="Scroll right"
                    onClick={() => scrollByAmount(1)}
                    whileTap={{ scale: 0.95 }}
                    className="absolute right-0 -top-0.5 z-10 bg-primary-blue/20 active:bg-primary-blue/40 
                             cursor-pointer w-8.5 h-14.5 border-none rounded-tl-full rounded-bl-full 
                             flex items-center justify-center"
                >
                    <ChevronRight className="text-white w-6 h-6 cursor-pointer ml-1" strokeWidth={0.9} />
                </motion.button>
            )}
        </div>
    );
}
