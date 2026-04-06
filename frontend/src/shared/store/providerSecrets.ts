import { create } from "zustand";
import { persist } from "zustand/middleware";

interface ProviderSecretStore {
  secrets: Record<string, string>;
  setSecret: (providerId: string, apiKey: string) => void;
}

export const useProviderSecrets = create<ProviderSecretStore>()(
  persist(
    (set) => ({
      secrets: {},
      setSecret: (providerId, apiKey) =>
        set((state) => ({
          secrets: {
            ...state.secrets,
            [providerId]: apiKey,
          },
        })),
    }),
    {
      name: "provider-secrets",
    }
  )
);
