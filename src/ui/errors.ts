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
      alert('Failed to read partition table, did you select a Nand dump?');
      return null;
    case NandError.InvalidProdKeys:
      alert('Invalid prod.keys provided, please ensure the prod.keys match the data you are using!');
      return null;
    case NandError.NoNandOpened:
      alert(`Failed to ${actionDescription} because no NAND file has been opened yet!`);
      return null;
    case NandError.NoPartitionMounted:
      alert(`Failed to ${actionDescription} because no partition has been mounted yet!`);
      return null;
    case NandError.NoProdKeys:
      alert(`Failed to ${actionDescription} because prod.keys are required but none found!`);
      return null;
    case NandError.Readonly:
      alert(`Failed to ${actionDescription} the NAND was opened in readonly mode!`);
      return null;
    case NandError.Unknown:
      console.error(result);
      alert(`An unknown error occurred while ${actionDescription}!`);
      return null;
  }
}
