/** Legacy wire identifiers are confined to this compatibility boundary. */
export const legacyClientHeader: string = 'X-Jarvis-Client';
export const legacyOperationHeader: string = 'X-Jarvis-Op-Id';
export function isPairingScheme(value: string): boolean { return value === 'persona:' || value === 'jarvis:'; }
export function isPairingAddress(value: string): boolean { return value.startsWith('persona:') || value.startsWith('jarvis:'); }
export function isPairingLink(value: string): boolean { return value.startsWith('persona://pair') || value.startsWith('jarvis://pair'); }

/** Normalize supported Persona deep links at the compatibility boundary. */
export function canonicalPersonaLink(value: string): string {
  const colon = value.indexOf(':');
  if (colon < 0 || !isPairingScheme(value.slice(0, colon).toLowerCase() + ':')) return '';
  return 'persona:' + value.slice(colon + 1);
}

export const subscriptionsPageId = '-subscriptions';
export const subscriptionsPageUI = 'pages';

export interface InboxSourcesLink {
  kind: 'ask' | 'graph';
  text: string;
}

/** Only the manager-drawn subscriptions page dispatches app-owned presentation links. */
export function inboxSourcesLink(address: string): InboxSourcesLink | undefined {
  const link = /^persona:\/\/([^/?#]+)\/?(?:\?([^#]*))?(?:#.*)?$/i.exec(canonicalPersonaLink(address));
  if (!link) return undefined;
  const host = link[1].toLowerCase();
  if (host === 'sources-graph') return { kind: 'graph', text: '' };
  if (host !== 'ask') return undefined;
  for (const item of (link[2] || '').split('&')) {
    const equals = item.indexOf('=');
    if (equals < 0 || item.slice(0, equals) !== 'text') continue;
    try {
      const text = decodeURIComponent(item.slice(equals + 1));
      if (text.trim() && Array.from(text).length <= 8000) return { kind: 'ask', text };
    } catch (_) { return undefined; }
    return undefined;
  }
  return undefined;
}
