import { useQuery } from "@tanstack/react-query"
import { Call } from "@wailsio/runtime"

import {
  normalizeFontCatalog,
  type FontCatalogFamily,
} from "@/shared/fonts/fontCatalog"

export const FONT_CATALOG_QUERY_KEY = ["fontCatalog"]

export function useFontCatalog() {
  return useQuery({
    queryKey: FONT_CATALOG_QUERY_KEY,
    queryFn: async (): Promise<FontCatalogFamily[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.SystemHandler.ListFontCatalog",
      )
      return normalizeFontCatalog(result)
    },
    staleTime: Infinity,
  })
}
