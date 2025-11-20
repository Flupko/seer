

import { getFeaturedCategories } from "@/lib/api";
import { MarketSearchSchema, sortOptions, statusOptions } from "@/lib/definitions";
import { redirect } from "next/navigation";
import MarketsHome from "./markets";

export default async function HomePage({ searchParams }: { searchParams: Promise<{ category?: string, sort?: string, status?: string }> }) {

  const categories = await getFeaturedCategories();
  const sp = await searchParams;

  const categorySlug = sp.category ?? categories[0].slug;
  const sort = sp.sort;
  const status = sp.status;

  const category = categories.find(c => c.slug === categorySlug);
  const sortValid = !sort || sortOptions.some(o => o.value === sort);
  const statusValid = !status || statusOptions.some(o => o.value === status);

  if (!category || !sortValid || !statusValid) {
    return redirect('/');
  }

  const parsed = MarketSearchSchema.safeParse({
    categorySlug,
    sort,
    status,
    pageSize: 8,
    page: 1,
  });

  const search = parsed.data;

  if (!parsed.success || !search) {
    return
  }


  return (
    <MarketsHome search={search} activeCategory={category} />
  )


}
