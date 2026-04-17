import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../index';

describe( 'schema v3 -> v4 migration', () => {
	let editor: ClassicEditor;
	let logSpy: ReturnType<typeof vi.spyOn>;
	let host: HTMLElement;

	beforeEach( async () => {
		logSpy = vi.spyOn( console, 'info' ).mockImplementation( () => {} );
		host = document.createElement( 'div' );
		document.body.appendChild( host );
		editor = await ClassicEditor.create( host, {
			licenseKey: 'GPL',
			plugins: [ Essentials, Paragraph, MddmBlockIdentityPlugin ],
		} );
	} );

	afterEach( async () => {
		logSpy.mockRestore();
		await editor.destroy();
		host.remove();
	} );

	it( 'mints bids on legacy content and logs upgrade', () => {
		editor.setData( '<p>legacy1</p><p>legacy2</p>' );
		const html = editor.getData();
		const matches = html.match( /data-mddm-bid="[0-9a-f-]{36}"/g ) ?? [];
		expect( matches ).toHaveLength( 2 );
		expect( logSpy ).toHaveBeenCalledWith( expect.stringContaining( 'schema-upgrade-v4' ) );
	} );

	it( 'no log for already-v4 content', () => {
		editor.setData( '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>' );
		expect( logSpy ).not.toHaveBeenCalledWith( expect.stringContaining( 'schema-upgrade-v4' ) );
	} );
} );
