import { NandResult } from '../channels';

export function handleNandResult<T>(result: NandResult<T>, actionDescription: string): T | null {
  if (result.type === 'success') {
    return result.data;
  }

  let message = `Failed to ${actionDescription}.\n\n${result.error}`;
  if (!message.endsWith('.')) message += '.';
  alert(message);
  return null;
}
