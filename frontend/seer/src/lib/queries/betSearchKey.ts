import { UserBetSearch } from "../definitions";

export const betSearchKey = (search?: UserBetSearch) =>
    search ? ['userBets', 'query', 'marketId', search.marketId ?? 'all', 'status', search.status ?? 'all', 'sort', search.sort, 'sortDir', search.sortDir, 'pageSize', search.pageSize, 'page', search.page] : ['userBets', 'invalid'];