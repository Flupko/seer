import { getFeaturedCategories } from "@/lib/api";
import BetsLive from "@/ui/bet/bets_live/BetsLive";
import MainWrapper from "@/ui/MainWrapper";

export default async function Layout({ children }: { children: React.ReactNode }) {
    const categories = await getFeaturedCategories();

    return (
        <>
            <MainWrapper>
                {children}
                <div className="pt-10 md:pt-14">
                    <BetsLive />
                </div>
            </MainWrapper>
        </>
    );
}