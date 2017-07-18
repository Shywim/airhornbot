interface Sound {
  id: string,
  name: string,
  gif: string,
  command: string,
  weight: number,
}

interface Guild {
  id: string,
  name: string,
  plays: number,
  sounds: Sound[],
}

interface GuildsResponse {
  airhorn: Guild[],
  boring: Guild[],
}