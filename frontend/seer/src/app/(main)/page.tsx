

import { getFeaturedCategories } from "@/lib/api";
import { sortOptions, statusOptions } from "@/lib/definitions";
import { Header } from "@/ui/markets/home/Header";
import MarketsDisplay from "@/ui/markets/home/MarketsDisplay";
import { redirect } from "next/navigation";

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

  return (
    <div className="pt-4 lg:pt-6 transition-all">
      <Header activeCategory={category} />
      <MarketsDisplay categories={categories} />
    </div>
  )


}
