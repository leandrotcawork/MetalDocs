import { Plugin } from 'ckeditor5';

export class MddmTableVariantPlugin extends Plugin {
	public static get pluginName() {
		return 'MddmTableVariantPlugin' as const;
	}

	public init(): void {
		const { editor } = this;

		editor.model.schema.extend('table', {
			allowAttributes: [ 'mddmTableVariant' ]
		});

		editor.conversion.attributeToAttribute({
			model: { name: 'table', key: 'mddmTableVariant' },
			view: 'data-mddm-variant',
		});
	}
}
