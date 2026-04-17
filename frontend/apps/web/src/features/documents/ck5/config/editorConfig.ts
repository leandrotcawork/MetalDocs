import type { EditorConfig } from 'ckeditor5';
import { AUTHOR_PLUGINS, FILL_PLUGINS } from './pluginLists';
import { AUTHOR_TOOLBAR, FILL_TOOLBAR } from './toolbars';
import { MddmFieldPlugin } from '../plugins/MddmFieldPlugin';
import { MddmSectionPlugin } from '../plugins/MddmSectionPlugin';
import { MddmRepeatablePlugin } from '../plugins/MddmRepeatablePlugin';
import { MddmDataTablePlugin } from '../plugins/MddmDataTablePlugin';
import { MddmRichBlockPlugin } from '../plugins/MddmRichBlockPlugin';
import { MddmUploadAdapterPlugin } from '../plugins/MddmUploadAdapter';
import { MddmBlockIdentityPlugin } from '../plugins/MddmBlockIdentityPlugin';
import { MddmPaginationPlugin } from '../plugins/MddmPaginationPlugin';

type PluginCtor = NonNullable<EditorConfig['plugins']>[number];

export interface ConfigOptions {
  language?: string;
  extraPlugins?: PluginCtor[];
  uploadEndpoint?: string;
  getAuthHeader?: () => string | null;
}

export function createAuthorConfig(opts: ConfigOptions = {}): EditorConfig {
  return {
    licenseKey: 'GPL',
    language: opts.language ?? 'en',
    plugins: [
      ...AUTHOR_PLUGINS,
      MddmBlockIdentityPlugin,
      MddmPaginationPlugin,
      MddmFieldPlugin,
      MddmSectionPlugin,
      MddmRepeatablePlugin,
      MddmDataTablePlugin,
      MddmUploadAdapterPlugin,
      MddmRichBlockPlugin,
      ...(opts.extraPlugins ?? []),
    ],
    toolbar: { items: [...AUTHOR_TOOLBAR] },
    image: {
      toolbar: [
        'imageTextAlternative',
        'imageStyle:inline',
        'imageStyle:block',
        'imageStyle:side',
        'toggleImageCaption',
        'resizeImage',
      ],
    },
    table: {
      contentToolbar: [
        'tableColumn',
        'tableRow',
        'mergeTableCells',
        'tableProperties',
        'tableCellProperties',
        'toggleTableCaption',
      ],
    },
    htmlSupport: {
      allow: [
        {
          name: /^(section|div|span|header|ol|li)$/,
          classes: (className: string) =>
            className.startsWith('mddm-') ||
            className.startsWith('restricted-editing-exception'),
          attributes: {
            'data-section-id': true,
            'data-variant': ['locked', 'editable', 'mixed'],
            'data-repeatable-id': true,
            'data-item-id': true,
            'data-field-id': true,
            'data-field-type': true,
            'data-field-label': true,
            'data-field-required': ['true', 'false'],
            'data-mddm-variant': ['fixed', 'dynamic'],
            'data-mddm-schema': (value: string) => /^v\d+$/.test(value),
          },
        },
      ],
      disallow: [
        { name: 'span', classes: 'mddm-field' },
      ],
    },
    // Read by MddmUploadAdapterPlugin; endpoint + auth header supplied by
    // callers (AuthorPage/FillPage) via the ConfigOptions passthrough.
    mddmUpload: opts.uploadEndpoint
      ? {
          endpoint: opts.uploadEndpoint,
          getAuthHeader: opts.getAuthHeader ?? (() => null),
        }
      : { endpoint: '/assets', getAuthHeader: () => null },
  } as unknown as EditorConfig;
}

export function createFillConfig(opts: ConfigOptions = {}): EditorConfig {
  const base = createAuthorConfig(opts);
  return {
    ...base,
    plugins: [
      ...FILL_PLUGINS,
      MddmBlockIdentityPlugin,
      MddmPaginationPlugin,
      MddmFieldPlugin,
      MddmSectionPlugin,
      MddmRepeatablePlugin,
      MddmDataTablePlugin,
      MddmRichBlockPlugin,
      MddmUploadAdapterPlugin,
      ...(opts.extraPlugins ?? []),
    ],
    toolbar: { items: [...FILL_TOOLBAR] },
    restrictedEditing: {
      allowedCommands: [
        'bold',
        'italic',
        'underline',
        'link',
        'alignment',
        'fontColor',
        'fontBackgroundColor',
      ],
      allowedAttributes: ['bold', 'italic', 'underline', 'linkHref'],
    },
  };
}
