import { getFeaturedCategories } from "@/lib/api";
import { MarketSearchSchema, sortOptions, statusOptions } from "@/lib/definitions";
import { redirect } from "next/navigation";
import MarketsSearch from "./markets";

export default async function SearchPage({ searchParams }: { searchParams: Promise<{ q?: string, category?: string, sort?: string, status?: string }> }) {

    const categories = await getFeaturedCategories();
    const sp = await searchParams;

    const q = sp.q;
    const categorySlug = sp.category;
    const sort = sp.sort;
    const status = sp.status;

    const category = !categorySlug || categories.find(c => c.slug === categorySlug);
    const sortValid = !sort || sortOptions.some(o => o.value === sort);
    const statusValid = !status || statusOptions.some(o => o.value === status);

    if (!category || !sortValid || !statusValid) {
        return redirect('/');
    }

    const parsed = MarketSearchSchema.safeParse({
        query: q ?? undefined,
        categorySlug: categorySlug ?? undefined,
        sort,
        status,
        pageSize: 6,
        page: 1,
    });

    const search = parsed.data;

    if (!parsed.success || !search) {
        return
    }


    return (
        <MarketsSearch search={search} />
    )
}