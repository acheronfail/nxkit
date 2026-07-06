import { NandResult } from '../channels';

export function handleNandResult<T>(result: NandResult<T>, actionDescription: string): T | null {
  if (result.type === 'success') {
    return result.data;
  }

  alert(`${actionDescription} failed!\n\n${result.error}`);
  return null;
}
