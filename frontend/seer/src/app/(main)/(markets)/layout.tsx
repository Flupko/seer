import { getFeaturedCategories } from "@/lib/api";
import CategoriesMenu from "@/ui/markets/home/categories/CategoriesMenu";

export default async function Layout({ children }: { children: React.ReactNode }) {
    const categories = await getFeaturedCategories();

    return (
        <>
            <CategoriesMenu categories={categories} />
            {children}
        </>
    );
}