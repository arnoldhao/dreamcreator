import { useQuery } from '@tanstack/react-query';
import { Call } from '@wailsio/runtime';

export const FONT_FAMILIES_QUERY_KEY = ['fontFamilies'];

export function useFontFamilies() {
  return useQuery({
    queryKey: FONT_FAMILIES_QUERY_KEY,
    queryFn: async (): Promise<string[]> => {
      const result = await Call.ByName(
        'dreamcreator/internal/presentation/wails.SystemHandler.ListFontFamilies'
      );
      if (!result) {
        return [];
      }
      return result as string[];
    },
    staleTime: Infinity,
  });
}
