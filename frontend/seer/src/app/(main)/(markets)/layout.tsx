import { getFeaturedCategories } from "@/lib/api";
import MainWrapper from "@/ui/MainWrapper";
import CategoriesMenu from "@/ui/markets/home/categories/CategoriesMenu";

export default async function Layout({ children }: { children: React.ReactNode }) {
    const categories = await getFeaturedCategories();

    return (
        <>
            <MainWrapper>
                <CategoriesMenu categories={categories} />
                {children}
            </MainWrapper>
        </>
    );
}