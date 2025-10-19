"use client";

import { Category } from "@/lib/definitions";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { motion } from "motion/react"; // framer-motion v11+ entry
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useRef, useState } from "react";
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
        const { clientWidth } = el;
        const amount = Math.round(clientWidth * 1) * dir;
        el.scrollBy({ left: amount, behavior: "smooth" });
    };

    const handleScroll = () => {
        if (!ref.current) return;
        const { scrollLeft, scrollWidth, clientWidth } = ref.current;
        switch (scrollLeft + clientWidth) {
            //Scroll is utter left
            case clientWidth:
                setDisableButton("left");
                break;
            //Scroll is utter right
            case scrollWidth:
                setDisableButton("right");
                break;
            //Scroll somewhere in between
            default:
                setDisableButton("none");
        }
    }

    const activeCategorySlug = useSearchParams().get("category") ?? categories[0]?.slug;

    return (
        <div className="relative">

            {disableButton !== "left" && (
                <motion.button
                    type="button"
                    aria-label="Scroll left"
                    onClick={() => scrollByAmount(-1)}
                    whileTap={{ scale: 0.95 }}
                    className="absolute left-0 -top-0.5 z-10 bg-primary-blue/20 active:bg-primary-blue/40 cursor-pointer w-8.5 h-14.5 border-none rounded-tr-full rounded-br-full flex items-center justify-center"
                >
                    <ChevronLeft className="text-white w-6 h-6 cursor-pointer mr-1" strokeWidth={0.9} />
                </motion.button>
            )}



            <div
                ref={ref}
                onScroll={handleScroll}
                className="flex overflow-x-auto hide-scrollbar gap-4 px-1"
            >
                {/* Ensure children snap by aligning each direct child */}
                {categories.map((category) => {

                    // URL Search paras
                    const searchParams = new URLSearchParams();
                    searchParams.set("category", category.slug);

                    return <Link href={`/?${searchParams.toString()}`} key={category.id} className="contents" prefetch={false}>
                        <CategoryButton
                            category={category}
                            active={category.slug === activeCategorySlug}
                        />
                    </Link>

                })}
            </div>

            {disableButton !== "right" && (
                <motion.button
                    type="button"
                    aria-label="Scroll right"
                    onClick={() => scrollByAmount(1)}
                    whileTap={{ scale: 0.95 }}
                    className="absolute right-0 -top-0.5 z-10 bg-primary-blue/20 active:bg-primary-blue/40 cursor-pointer w-8.5 h-14.5 border-none rounded-tl-full rounded-bl-full flex items-center justify-center"
                >
                    <ChevronRight className="text-white w-6 h-6 cursor-pointer ml-1" strokeWidth={0.9} />
                </motion.button>
            )}



        </div>
    );
}
