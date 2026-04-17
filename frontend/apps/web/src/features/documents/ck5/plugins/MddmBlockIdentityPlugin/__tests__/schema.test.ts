import { afterEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Heading, Table } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../index';

describe( 'MddmBlockIdentityPlugin', () => {
	const editors: Array<ClassicEditor> = [];
	const hosts: Array<HTMLElement> = [];

	afterEach( async () => {
		while ( editors.length ) {
			const editor = editors.pop();
			await editor?.destroy();
		}

		while ( hosts.length ) {
			hosts.pop()?.remove();
		}
	} );

	async function createEditor(): Promise<ClassicEditor> {
		const el = document.createElement( 'div' );
		document.body.appendChild( el );
		hosts.push( el );

		const editor = await ClassicEditor.create( el, {
			licenseKey: 'GPL',
			plugins: [ Essentials, Paragraph, Heading, Table, MddmBlockIdentityPlugin ],
		} );
		editors.push( editor );
		return editor;
	}

	it( 'registers pluginName as MddmBlockIdentity', () => {
		expect( MddmBlockIdentityPlugin.pluginName ).toBe( 'MddmBlockIdentity' );
	} );

	it( 'allows mddmBid attribute on registered paginable elements', async () => {
		const editor = await createEditor();
		const schema = editor.model.schema;

		const expected = [ 'paragraph', 'heading1', 'heading2', 'heading3', 'tableRow' ];
		for ( const name of expected ) {
			expect( schema.isRegistered( name ), `${ name } should be registered` ).toBe( true );
			expect( schema.checkAttribute( [ name ], 'mddmBid' ), `${ name } should allow mddmBid` ).toBe( true );
		}
	} );

	it( 'skips elements that are not registered', async () => {
		// Without BlockQuote/Image plugins, these should not be registered and
		// extendSchemaWithBid must not throw.
		const editor = await createEditor();
		const schema = editor.model.schema;

		expect( schema.isRegistered( 'blockQuote' ) ).toBe( false );
		expect( schema.isRegistered( 'imageBlock' ) ).toBe( false );
	} );
} );
