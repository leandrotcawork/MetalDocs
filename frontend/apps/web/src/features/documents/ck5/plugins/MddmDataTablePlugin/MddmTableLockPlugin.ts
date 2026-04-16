import { Plugin } from 'ckeditor5';
import { findAncestorByName } from '../../shared/findAncestor';

const STRUCTURAL_COMMANDS = [
	'insertTableRowAbove',
	'insertTableRowBelow',
	'insertTableColumnLeft',
	'insertTableColumnRight',
	'removeTableRow',
	'removeTableColumn',
	'mergeTableCells',
	'splitTableCellVertically',
	'splitTableCellHorizontally',
	'mergeTableCellRight',
	'mergeTableCellDown',
	'mergeTableCellLeft',
	'mergeTableCellUp',
	'setTableColumnHeader',
	'setTableRowHeader'
] as const;

export class MddmTableLockPlugin extends Plugin {
	public static get pluginName() {
		return 'MddmTableLockPlugin' as const;
	}

	public init(): void {
		const refreshCommands = (): void => {
			const position = this.editor.model.document.selection.getFirstPosition();
			const table = position ? findAncestorByName( position.parent, 'table' ) : null;
			const isFixedTable =
				!!table &&
				table.is( 'element', 'table' ) &&
				table.getAttribute( 'mddmTableVariant' ) === 'fixed';

			for ( const commandName of STRUCTURAL_COMMANDS ) {
				const command = this.editor.commands.get( commandName );

				if ( !command ) {
					continue;
				}

				if ( isFixedTable ) {
					command.forceDisabled( 'mddmTableLock' );
				} else {
					command.clearForceDisabled( 'mddmTableLock' );
				}
			}
		};

		this.listenTo( this.editor.model.document.selection, 'change:range', refreshCommands );
		this.listenTo( this.editor.model.document, 'change:data', refreshCommands );
		refreshCommands();
	}
}
