import { getFeaturedCategories } from "@/lib/api";
import BetsLive from "@/ui/bet/bets_live/BetsLive";
import MainWrapper from "@/ui/MainWrapper";

export default async function Layout({ children }: { children: React.ReactNode }) {
    const categories = await getFeaturedCategories();

    return (
        <>
            {children}
            <div className="pt-5 md:pt-8">
                <MainWrapper>
                    <BetsLive />
                </MainWrapper>
            </div>
        </>
    );
}