export const phoneAppearancePages: string[] = [
  'phone_look_color', 'phone_look_style', 'phone_look_navigation', 'phone_look_layout', 'phone_look_scheme'
];

export function phoneAppearancePage(page: string): boolean {
  return page === 'appearance' || page === 'phone_look_avatar' || phoneAppearancePages.includes(page);
}

export function phoneSettingsLink(page: string): string {
  if (page === 'phone_agent_layout') return 'phone_look_layout';
  if (phoneAppearancePage(page) || ['phone_agent_options', 'phone_language_options', 'phone_meeting_options', 'phone_knowledge_selection'].includes(page)) return page;
  return '';
}

export function pushPhoneSetting(trail: string[], link: string): string[] {
  const page = phoneSettingsLink(link);
  return !page || trail[trail.length - 1] === page ? trail : trail.concat([page]);
}
