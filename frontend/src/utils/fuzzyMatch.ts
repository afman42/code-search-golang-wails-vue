export function fuzzyMatch(text: string, query: string): boolean {
  if (!query || !text) return false;
  
  const cleanQuery = query.toLowerCase().replace(/\s+/g, '');
  const cleanText = text.toLowerCase();
  
  if (cleanQuery.length > cleanText.length) return false;
  
  let queryIndex = 0;
  let textIndex = 0;
  let matchFound = 0;
  
  while (textIndex < cleanText.length && queryIndex < cleanQuery.length) {
    if (cleanText[textIndex] === cleanQuery[queryIndex]) {
      matchFound++;
      queryIndex++;
    }
    textIndex++;
  }
  
  return matchFound >= Math.floor(cleanQuery.length * 0.8);
}

export function findFuzzyMatches(text: string, query: string): Array<{
  start: number;
  end: number;
  matchedChars: string[];
}> {
  const matches: Array<{ start: number; end: number; matchedChars: string[] }> = [];
  if (!query || !text) return matches;
  
  const cleanQuery = query.toLowerCase();
  const lowerText = text.toLowerCase();
  
  let pos = 0;
  while (pos <= lowerText.length - query.length) {
    const segment = lowerText.substring(pos, pos + query.length);
    let sameCount = 0;
    const matchedChars: string[] = [];
    
    for (let i = 0; i < query.length; i++) {
      if (segment[i] === cleanQuery[i]) {
        sameCount++;
        matchedChars.push(segment[i]);
      }
    }
    
    if (sameCount >= Math.floor(query.length * 0.6)) {
      matches.push({
        start: pos,
        end: pos + query.length,
        matchedChars,
      });
    }
    
    pos++;
  }
  
  return matches;
}

export function normalizeQuery(query: string): string {
  return query
    .toLowerCase()
    .trim()
    .replace(/\s+/g, ' ')
    .replace(/[^\w\s\-./]/g, '');
}
