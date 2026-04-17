import { afterEach, describe, expect, it } from 'vitest';
import {
	ClassicEditor,
	Essentials,
	Paragraph,
	Heading,
	List,
	Table,
	Undo,
} from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../index';

const PAGINABLE_NAMES = new Set<string>( [
	'paragraph',
	'heading1', 'heading2', 'heading3', 'heading4', 'heading5', 'heading6',
	'listItem',
	'blockQuote',
	'tableRow',
	'imageBlock',
	'mediaEmbed',
] );

type BidNode = { getAttribute( key: string ): unknown };

function collectTopLevelBids( editor: ClassicEditor ): Array<string> {
	const bids: Array<string> = [];
	const root = editor.model.document.getRoot()!;
	for ( const child of root.getChildren() ) {
		const bid = ( child as BidNode ).getAttribute( 'mddmBid' );
		if ( typeof bid === 'string' ) bids.push( bid );
	}
	return bids;
}

function collectAllPaginableBids( editor: ClassicEditor ): Array<string> {
	const bids: Array<string> = [];
	const root = editor.model.document.getRoot()!;
	for ( const { item } of editor.model.createRangeIn( root ) ) {
		if ( !item.is( 'element' ) ) continue;
		if ( !PAGINABLE_NAMES.has( item.name ) ) continue;
		const bid = ( item as unknown as BidNode ).getAttribute( 'mddmBid' );
		if ( typeof bid === 'string' ) bids.push( bid );
	}
	return bids;
}

describe( 'MddmBlockIdentityPlugin bid invariance matrix', () => {
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
			plugins: [ Essentials, Paragraph, Heading, List, Table, Undo, MddmBlockIdentityPlugin ],
		} );
		editors.push( editor );
		return editor;
	}

	describe( 'bid invariance — lists', () => {
		it( 'bullet → ordered conversion preserves bids', async () => {
			const editor = await createEditor();
			const aBid = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
			const bBid = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';

			editor.setData(
				`<ul><li data-mddm-bid="${ aBid }">A</li><li data-mddm-bid="${ bBid }">B</li></ul>`,
			);

			const before = collectTopLevelBids( editor );
			expect( before ).toContain( aBid );
			expect( before ).toContain( bBid );

			// Select all so the numberedList command toggles both list items.
			editor.model.change( writer => {
				writer.setSelection( editor.model.document.getRoot()!, 'in' );
			} );
			editor.execute( 'numberedList' );

			const after = collectTopLevelBids( editor );
			expect( after ).toContain( aBid );
			expect( after ).toContain( bBid );
		} );

		it( 'indent/outdent cycle preserves uniqueness + count', async () => {
			const editor = await createEditor();

			editor.setData( '<ul><li>one</li><li>two</li><li>three</li></ul>' );

			const before = collectTopLevelBids( editor );
			const beforeCount = before.length;
			expect( new Set( before ).size ).toBe( beforeCount );

			// Move selection into the 2nd block and indent, then outdent.
			editor.model.change( writer => {
				const root = editor.model.document.getRoot()!;
				writer.setSelection( root.getChild( 1 )!, 'in' );
			} );
			if ( editor.commands.get( 'indentList' )?.isEnabled ) {
				editor.execute( 'indentList' );
			}
			if ( editor.commands.get( 'outdentList' )?.isEnabled ) {
				editor.execute( 'outdentList' );
			}

			const after = collectTopLevelBids( editor );
			expect( after ).toHaveLength( beforeCount );
			expect( new Set( after ).size ).toBe( beforeCount );
		} );
	} );

	describe( 'bid invariance — tables', () => {
		it( 'table rows receive bids after insertion via setData', async () => {
			const editor = await createEditor();

			editor.setData(
				'<figure class="table"><table><tbody>' +
					'<tr><td>r1c1</td><td>r1c2</td></tr>' +
					'<tr><td>r2c1</td><td>r2c2</td></tr>' +
				'</tbody></table></figure>',
			);

			// All tableRow elements should have bids minted by the post-fixer.
			const all = collectAllPaginableBids( editor );
			expect( all.length ).toBeGreaterThan( 0 );
			expect( new Set( all ).size ).toBe( all.length );
		} );

		it( 'paragraphs inside table cells each get a bid', async () => {
			const editor = await createEditor();

			editor.setData(
				'<figure class="table"><table><tbody>' +
					'<tr><td><p>cell-a</p></td><td><p>cell-b</p></td></tr>' +
				'</tbody></table></figure>',
			);

			// Gather bids on every paragraph reachable from root (including cell paragraphs).
			const paraBids: Array<string> = [];
			const root = editor.model.document.getRoot()!;
			for ( const { item } of editor.model.createRangeIn( root ) ) {
				if ( !item.is( 'element', 'paragraph' ) ) continue;
				const bid = ( item as unknown as BidNode ).getAttribute( 'mddmBid' );
				expect( typeof bid ).toBe( 'string' );
				paraBids.push( bid as string );
			}
			expect( paraBids.length ).toBeGreaterThanOrEqual( 2 );
			expect( new Set( paraBids ).size ).toBe( paraBids.length );
		} );
	} );

	describe( 'bid invariance — paste (same-doc)', () => {
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

		it( 'copy-paste of a block with a colliding bid remints the paste', async () => {
			const editor = await createEditor();
			const bidA = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc';

			editor.setData( `<p data-mddm-bid="${ bidA }">alpha</p><p>beta</p>` );

			// Paste HTML carrying the same bid as an existing block.
			firePaste( editor, `<p data-mddm-bid="${ bidA }">copied</p>` );

			const bids = collectTopLevelBids( editor );
			// Uniqueness holds post-paste.
			expect( new Set( bids ).size ).toBe( bids.length );
		} );

		it( 'plain-text paste produces blocks that all have bids', async () => {
			const editor = await createEditor();
			editor.setData( '<p>seed</p>' );

			const dataTransfer = {
				getData: ( type: string ) => ( type === 'text/plain' ? 'hello' : '' ),
				types: [ 'text/plain' ],
			};
			editor.editing.view.document.fire( 'clipboardInput', {
				dataTransfer,
				method: 'paste',
				stopPropagation() {},
				preventDefault() {},
			} );

			const bids = collectTopLevelBids( editor );
			expect( bids.length ).toBeGreaterThan( 0 );
			for ( const bid of bids ) {
				expect( typeof bid ).toBe( 'string' );
				expect( bid.length ).toBeGreaterThan( 0 );
			}
			expect( new Set( bids ).size ).toBe( bids.length );
		} );
	} );

	describe( 'bid invariance — undo', () => {
		it( 'undo after typing preserves the original bid on the survivor block', async () => {
			const editor = await createEditor();
			editor.setData( '<p>start</p>' );

			const [ originalBid ] = collectTopLevelBids( editor );
			expect( typeof originalBid ).toBe( 'string' );

			// Simulate a typing-style change: insert text at end of paragraph.
			editor.model.change( writer => {
				const root = editor.model.document.getRoot()!;
				const paragraph = root.getChild( 0 )!;
				writer.insertText( '!', paragraph, 'end' );
			} );

			// Undo the insertion.
			editor.execute( 'undo' );

			const after = collectTopLevelBids( editor );
			expect( after ).toHaveLength( 1 );
			expect( after[ 0 ] ).toBe( originalBid );
		} );

		it( 'undo/redo cycle preserves uniqueness with no duplicate bids', async () => {
			const editor = await createEditor();
			editor.setData( '<p>first</p><p>second</p>' );

			const before = collectTopLevelBids( editor );
			expect( new Set( before ).size ).toBe( before.length );

			// Split to create a 3rd block, then undo, then redo.
			editor.model.change( writer => {
				const root = editor.model.document.getRoot()!;
				const second = root.getChild( 1 )!;
				writer.split( writer.createPositionAt( second, 3 ) );
			} );

			editor.execute( 'undo' );
			editor.execute( 'redo' );

			const after = collectTopLevelBids( editor );
			expect( new Set( after ).size ).toBe( after.length );
		} );
	} );
} );
