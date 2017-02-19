type MessageEvent = {
  data: string
}

type MessageEventData = {
  total: number,
  unique_users: number,
  unique_guilds: number,
  unique_channels: number,
}

declare class EventSource {
  onmessage: (event: MessageEvent) => void;
}
