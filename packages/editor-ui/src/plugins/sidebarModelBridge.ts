import type { ReactEditorPlugin, ReactSidebarItem } from '@eigenpal/docx-js-editor';
import { createElement } from 'react';
import type { SidebarModel } from './mergefieldPlugin';

const SIDEBAR_PLUGIN_ID = 'metaldocs-sidebar-model';
const SIDEBAR_PLUGIN_NAME = 'metaldocs-sidebar-model';

function buildItem(id: string, title: string, values: string[]): ReactSidebarItem {
  return {
    id,
    anchorPos: 0,
    render: () =>
      createElement(
        'section',
        null,
        createElement('h4', null, title),
        createElement(
          'ul',
          null,
          values.map((value) => createElement('li', { key: value }, value))
        )
      ),
  };
}

export function buildSidebarModelPlugin(model: SidebarModel): ReactEditorPlugin {
  return {
    id: SIDEBAR_PLUGIN_ID,
    name: SIDEBAR_PLUGIN_NAME,
    getSidebarItems: () => {
      const items: ReactSidebarItem[] = [];

      if (model.used.length > 0) {
        items.push(buildItem('metaldocs-used-fields', 'Used fields', model.used));
      }
      if (model.missing.length > 0) {
        items.push(buildItem('metaldocs-missing-fields', 'Missing fields', model.missing));
      }
      if (model.orphans.length > 0) {
        items.push(buildItem('metaldocs-orphan-tokens', 'Orphan tokens', model.orphans));
      }
      if (model.bannerError) {
        items.push(buildItem('metaldocs-errors', 'Errors', model.errorCategories));
      }

      return items;
    },
  };
}
