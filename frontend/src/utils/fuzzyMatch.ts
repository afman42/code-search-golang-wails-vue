const SLIDING_WINDOW_SIMILARITY_THRESHOLD = 0.6;
const MAX_TEXT_LENGTH_FOR_FUZZY_SEARCH = 50000;

export function findFuzzyMatches(text: string, query: string): Array<{
  start: number;
  end: number;
  matchedChars: string[];
}> {
  const matches: Array<{ start: number; end: number; matchedChars: string[] }> = [];
  if (!query || !text) return matches;
  
  // Optimize for long texts to avoid O(n*m) performance issues
  if (text.length > MAX_TEXT_LENGTH_FOR_FUZZY_SEARCH) {
    return [];
  }
  
  const cleanQuery = query.toLowerCase();
  const lowerText = text.toLowerCase();
  
  const minPositions = Math.max(0, lowerText.length - query.length);
  
  let pos = 0;
  while (pos <= minPositions) {
    const segment = lowerText.substring(pos, pos + query.length);
    let sameCount = 0;
    const matchedChars: string[] = [];
    
    for (let i = 0; i < query.length; i++) {
      if (segment[i] === cleanQuery[i]) {
        sameCount++;
        matchedChars.push(segment[i]);
      }
    }
    
    if (sameCount >= Math.floor(query.length * SLIDING_WINDOW_SIMILARITY_THRESHOLD)) {
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

