import { afterEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../index';

describe( 'MddmBlockIdentityPlugin clipboard collision remint', () => {
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

	function firePaste( editor: ClassicEditor, html: string ): void {
		const dataTransfer = {
			getData: ( type: string ) => ( type === 'text/html' ? html : '' ),
			types: [ 'text/html' ],
		};
		editor.editing.view.document.fire( 'clipboardInput', {
			dataTransfer,
			method: 'paste',
			stopPropagation() {},
			preventDefault() {},
		} );
	}

	it( 're-mints paragraph bid that collides with an existing document bid', async () => {
		const editor = await createEditor();
		const collidingBid = '11111111-1111-4111-8111-111111111111';

		// Seed the document with a paragraph that owns the colliding bid.
		editor.setData( `<p data-mddm-bid="${ collidingBid }">original</p>` );

		// Paste HTML carrying the same bid.
		firePaste( editor, `<p data-mddm-bid="${ collidingBid }">pasted</p>` );

		const root = editor.model.document.getRoot()!;
		const bids = new Set<string>();
		let collidingCount = 0;
		for ( const node of root.getChildren() ) {
			const bid = ( node as { getAttribute( key: string ): unknown } ).getAttribute( 'mddmBid' ) as string | undefined;
			expect( typeof bid ).toBe( 'string' );
			bids.add( bid! );
			if ( bid === collidingBid ) collidingCount++;
		}

		// All bids must be unique (no duplicates after paste).
		expect( bids.size ).toBe( root.childCount );
		// The colliding bid must survive on at most one element (the original).
		expect( collidingCount ).toBeLessThanOrEqual( 1 );
	} );

	it( 'preserves a non-colliding pasted bid', async () => {
		const editor = await createEditor();
		const docBid = '22222222-2222-4222-8222-222222222222';
		const pastedBid = '33333333-3333-4333-8333-333333333333';

		// Seed with a paragraph carrying docBid.
		editor.setData( `<p data-mddm-bid="${ docBid }">original</p>` );
		// Paste a two-paragraph fragment; the second block will survive as its own element.
		firePaste(
			editor,
			`<p data-mddm-bid="${ pastedBid }">pasted-a</p><p data-mddm-bid="44444444-4444-4444-8444-444444444444">pasted-b</p>`,
		);

		const root = editor.model.document.getRoot()!;
		const bids = new Set<string>();
		for ( const node of root.getChildren() ) {
			const bid = ( node as { getAttribute( key: string ): unknown } ).getAttribute( 'mddmBid' ) as string;
			bids.add( bid );
		}
		// The non-colliding pasted bid should be preserved somewhere in the doc.
		expect( bids.has( pastedBid ) ).toBe( true );
	} );
} );
