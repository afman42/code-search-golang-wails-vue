// Recent-search persistence types.

// Single source of truth for a persisted search. `directory` is optional so
// entries written by older versions (query + extension only) keep working.
export interface RecentSearch {
  query: string;
  extension: string;
  directory?: string;
}
