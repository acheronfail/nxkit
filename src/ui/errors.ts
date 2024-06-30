import { NandError, NandResult } from '../channels';

export function handleNandResult<T>(result: NandResult<T>, actionDescription: string): T | null {
  switch (result.error) {
    case NandError.None: {
      if ('data' in result) {
        return result.data;
      }

      return null;
    }
    case NandError.InvalidPartitionTable:
      alert('Failed to read partition table.\n\nDid you select a NAND dump?');
      return null;
    case NandError.InvalidProdKeys:
      alert('Invalid prod.keys provided.\n\n Please ensure the prod.keys match the data you are using!');
      return null;
    case NandError.NoNandOpened:
      alert(`Failed to ${actionDescription}.\n\nNo NAND file has been opened yet!`);
      return null;
    case NandError.NoPartitionMounted:
      alert(`Failed to ${actionDescription}.\n\nNo partition has been mounted yet!`);
      return null;
    case NandError.NoProdKeys:
      alert(`Failed to ${actionDescription}.\n\nprod.keys are required but none found!`);
      return null;
    case NandError.Readonly:
      alert(`Failed to ${actionDescription}.\n\nThe NAND was opened in readonly mode!`);
      return null;
    case NandError.AlreadyExists:
      alert(`Failed to ${actionDescription}.\n\nAn item already exists with that name!`);
      return null;
    case NandError.Generic:
      console.error(result);
      alert(`Failed to ${actionDescription}.\n\n${result.description}`);
      return null;
  }
}
