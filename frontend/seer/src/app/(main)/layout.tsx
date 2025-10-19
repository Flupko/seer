import { getFeaturedCategories } from "@/lib/api";
import MainWrapper from "@/ui/MainWrapper";
import CategoriesMenu from "@/ui/markets/home/categories/CategoriesMenu";

export default async function Layout({ children, searchParams }: { children: React.ReactNode, searchParams: Promise<{ category?: string, sort?: string }> }) {

    const categories = await getFeaturedCategories();

    return (
        <>
            <MainWrapper>
                <CategoriesMenu categories={categories} />
                {children}
            </MainWrapper>
        </>
    )
}