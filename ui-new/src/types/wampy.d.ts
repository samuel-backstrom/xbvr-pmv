declare module 'wampy' {
  export interface WampyEvent {
    argsDict?: Record<string, any>
    argsList?: any[]
    details?: Record<string, any>
  }

  export interface WampyOptions {
    realm?: string
    onConnect?: () => void
    onClose?: () => void
    onError?: () => void
    onReconnect?: () => void
    onReconnectSuccess?: () => void
  }

  export class Wampy {
    constructor(url: string, options?: WampyOptions)
    subscribe(topic: string, callback: (event: WampyEvent) => void): Wampy
    disconnect(): void
  }

  export default Wampy
}
