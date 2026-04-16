import type { LayoutTokens } from "../layout-ir";

export type InterpretContext = {
  depth: number;
  sectionIndex?: number;
  parentNumber?: string;
  parentType?: string;
};

export interface BlockLayoutInterpreter<TBlock, TViewModel> {
  interpret(block: TBlock, tokens: LayoutTokens, context: InterpretContext): TViewModel;
}
