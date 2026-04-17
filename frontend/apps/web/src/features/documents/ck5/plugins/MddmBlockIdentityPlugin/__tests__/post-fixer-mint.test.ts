import { afterEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../index';

const UUID_V4_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

describe( 'MddmBlockIdentityPlugin bid post-fixer', () => {
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
			plugins: [ Essentials, Paragraph, MddmBlockIdentityPlugin ],
		} );
		editors.push( editor );
		return editor;
	}

	it( 'mints UUID v4 mddmBid for paragraph created from data', async () => {
		const editor = await createEditor();

		editor.setData( '<p>hello</p>' );

		const root = editor.model.document.getRoot()!;
		const paragraph = root.getChild( 0 );
		expect( paragraph?.is( 'element', 'paragraph' ) ).toBe( true );

		const bid = ( paragraph as { getAttribute( key: string ): unknown } ).getAttribute( 'mddmBid' );
		expect( typeof bid ).toBe( 'string' );
		expect( bid ).toMatch( UUID_V4_REGEX );
	} );

	it( 'does not remint when paragraph already has mddmBid', async () => {
		const editor = await createEditor();
		const presetBid = '11111111-1111-4111-8111-111111111111';

		editor.setData( '<p>hello</p>' );

		editor.model.change( writer => {
			const root = editor.model.document.getRoot()!;
			const paragraph = root.getChild( 0 )!;
			writer.setAttribute( 'mddmBid', presetBid, paragraph );
		} );

		editor.model.change( writer => {
			const root = editor.model.document.getRoot()!;
			const paragraph = root.getChild( 0 )!;
			writer.insertText( '!', paragraph, 'end' );
		} );

		const root = editor.model.document.getRoot()!;
		const paragraph = root.getChild( 0 )!;
		const bid = ( paragraph as { getAttribute( key: string ): unknown } ).getAttribute( 'mddmBid' );
		expect( bid ).toBe( presetBid );
	} );

	it( 'mints unique mddmBid values for 100 paragraphs inserted in one change', async () => {
		const editor = await createEditor();

		editor.model.change( writer => {
			const root = editor.model.document.getRoot()!;

			for ( let i = 0; i < 100; i++ ) {
				const paragraph = writer.createElement( 'paragraph' );
				writer.insert( paragraph, root, 'end' );
				writer.insertText( `line ${ i }`, paragraph, 0 );
			}
		} );

		const root = editor.model.document.getRoot()!;
		// Editor initializes with 1 default paragraph, so 100 appended = 101 total
		expect( root.childCount ).toBe( 101 );

		const bids = new Set<string>();
		for ( const node of root.getChildren() ) {
			const bid = ( node as { getAttribute( key: string ): unknown } ).getAttribute( 'mddmBid' );
			expect( typeof bid ).toBe( 'string' );
			bids.add( bid as string );
		}

		expect( bids.size ).toBe( 101 );
	} );
} );
