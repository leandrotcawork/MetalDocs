import { afterEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../index';

const UUID_V4_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

describe( 'MddmBlockIdentityPlugin split/merge semantics', () => {
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

	function bids( editor: ClassicEditor ): Array<string> {
		const root = editor.model.document.getRoot()!;
		const out: Array<string> = [];
		for ( const child of root.getChildren() ) {
			const bid = ( child as { getAttribute( key: string ): unknown } ).getAttribute( 'mddmBid' );
			out.push( bid as string );
		}
		return out;
	}

	it( 'split: survivor keeps bid, new block gets fresh bid', async () => {
		const editor = await createEditor();
		editor.setData( '<p>hello world</p>' );

		const before = bids( editor );
		expect( before ).toHaveLength( 1 );
		const originalBid = before[ 0 ];

		editor.model.change( writer => {
			const root = editor.model.document.getRoot()!;
			const paragraph = root.getChild( 0 )!;
			writer.split( writer.createPositionAt( paragraph, 5 ) );
		} );

		const after = bids( editor );
		expect( after ).toHaveLength( 2 );
		expect( after[ 0 ] ).toBe( originalBid );
		expect( after[ 1 ] ).not.toBe( originalBid );
		expect( after[ 1 ] ).toMatch( UUID_V4_REGEX );
	} );

	it( 'merge: survivor (earlier) bid stays, absorbed bid dropped', async () => {
		const editor = await createEditor();
		editor.setData( '<p>first</p><p>second</p>' );

		const before = bids( editor );
		expect( before ).toHaveLength( 2 );
		const firstBid = before[ 0 ];
		const secondBid = before[ 1 ];
		expect( firstBid ).not.toBe( secondBid );

		editor.model.change( writer => {
			const root = editor.model.document.getRoot()!;
			const second = root.getChild( 1 )!;
			writer.merge( writer.createPositionBefore( second ) );
		} );

		const after = bids( editor );
		expect( after ).toHaveLength( 1 );
		expect( after[ 0 ] ).toBe( firstBid );
	} );

	it( 'split then merge preserves original bid', async () => {
		const editor = await createEditor();
		editor.setData( '<p>hello world</p>' );

		const originalBid = bids( editor )[ 0 ];

		editor.model.change( writer => {
			const root = editor.model.document.getRoot()!;
			const paragraph = root.getChild( 0 )!;
			writer.split( writer.createPositionAt( paragraph, 5 ) );
		} );

		editor.model.change( writer => {
			const root = editor.model.document.getRoot()!;
			const second = root.getChild( 1 )!;
			writer.merge( writer.createPositionBefore( second ) );
		} );

		const after = bids( editor );
		expect( after ).toHaveLength( 1 );
		expect( after[ 0 ] ).toBe( originalBid );
	} );
} );
