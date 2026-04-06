import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  ClearConnectorRequest,
  ConnectConnectorRequest,
  Connector,
  OpenConnectorSiteRequest,
  UpsertConnectorRequest,
} from "@/shared/contracts/connectors";
import {
  ClearConnector as ClearConnectorBinding,
  ConnectConnector as ConnectConnectorBinding,
  InstallPlaywright,
  ListConnectors,
  OpenConnectorSite as OpenConnectorSiteBinding,
  UpsertConnector as UpsertConnectorBinding,
} from "../../../bindings/dreamcreator/internal/presentation/wails/connectorshandler";
import {
  ClearConnectorRequest as BindingsClearConnectorRequest,
  ConnectConnectorRequest as BindingsConnectConnectorRequest,
  Connector as BindingsConnector,
  OpenConnectorSiteRequest as BindingsOpenConnectorSiteRequest,
  UpsertConnectorRequest as BindingsUpsertConnectorRequest,
} from "../../../bindings/dreamcreator/internal/application/connectors/dto/models";

export const CONNECTORS_QUERY_KEY = ["connectors"];

export function useConnectors() {
  return useQuery({
    queryKey: CONNECTORS_QUERY_KEY,
    queryFn: async (): Promise<Connector[]> => {
      return (await ListConnectors()).map(toConnector);
    },
    staleTime: 5_000,
  });
}

export function useUpsertConnector() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: UpsertConnectorRequest): Promise<Connector> => {
      return toConnector(await UpsertConnectorBinding(BindingsUpsertConnectorRequest.createFrom(request)));
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CONNECTORS_QUERY_KEY });
    },
  });
}

export function useClearConnector() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: ClearConnectorRequest): Promise<void> => {
      await ClearConnectorBinding(BindingsClearConnectorRequest.createFrom(request));
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CONNECTORS_QUERY_KEY });
    },
  });
}

export function useConnectConnector() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: ConnectConnectorRequest): Promise<Connector> => {
      return toConnector(await ConnectConnectorBinding(BindingsConnectConnectorRequest.createFrom(request)));
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CONNECTORS_QUERY_KEY });
    },
  });
}

export function useOpenConnectorSite() {
  return useMutation({
    mutationFn: async (request: OpenConnectorSiteRequest): Promise<void> => {
      await OpenConnectorSiteBinding(BindingsOpenConnectorSiteRequest.createFrom(request));
    },
  });
}

export function useInstallPlaywright() {
  return useMutation({
    mutationFn: async (): Promise<void> => {
      await InstallPlaywright();
    },
  });
}

function toConnector(raw: BindingsConnector): Connector {
  return {
    ...raw,
    cookies: raw.cookies.map((item) => ({ ...item })),
  };
}
