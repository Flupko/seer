import { MarketSearch } from "../definitions";

export const marketSearchKey = (search?: MarketSearch) =>
    search ? ['market', 'query', search.query, 'category', search.categorySlug, 'sort', search.sort, 'status', search.status, 'pageSize', search.pageSize, 'page', search.page] : ['market', 'invalid'];