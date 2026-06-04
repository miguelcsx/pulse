import type {
  AskCreateInput,
  AskCreateResponse,
  Bridge,
  HelpSession,
  HelpSignal,
  HelpSignalKind,
  TodayResponse,
  TrustProfile,
  TrustProfileInput,
} from "@pulse/drift/types";
import client from "./client";

export async function getToday(): Promise<TodayResponse> {
  const response = await client.get<TodayResponse>("/today");
  return response.data;
}

export async function createAsk(input: AskCreateInput): Promise<AskCreateResponse> {
  const response = await client.post<AskCreateResponse>("/asks", input);
  return response.data;
}

export async function getAskBridges(askId: string): Promise<Bridge[]> {
  const response = await client.get<Bridge[]>(`/asks/${askId}/bridges`);
  return response.data;
}

export async function askBridge(bridgeId: string, message = ""): Promise<Bridge> {
  const response = await client.post<Bridge>(`/bridges/${bridgeId}/ask`, {
    message,
  });
  return response.data;
}

export async function respondBridge(
  bridgeId: string,
  message = "",
): Promise<Bridge> {
  const response = await client.post<Bridge>(`/bridges/${bridgeId}/respond`, {
    message,
  });
  return response.data;
}

export async function signalBridge(
  bridgeId: string,
  kind: HelpSignalKind,
): Promise<HelpSignal> {
  const response = await client.post<HelpSignal>(`/bridges/${bridgeId}/signal`, {
    kind,
  });
  return response.data;
}

export async function listHelpSessions(): Promise<HelpSession[]> {
  const response = await client.get<HelpSession[]>("/help-sessions");
  return response.data;
}

export async function joinHelpSession(sessionId: string): Promise<HelpSession> {
  const response = await client.post<HelpSession>(
    `/help-sessions/${sessionId}/join`,
  );
  return response.data;
}

export async function updateTrustProfile(
  input: TrustProfileInput,
): Promise<TrustProfile> {
  const response = await client.put<TrustProfile>("/me/trust-profile", input);
  return response.data;
}
