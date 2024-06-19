import type { ProdKeys } from '../../channels';

function createKeyStore() {
  let keysFromMain = $state<ProdKeys | undefined>(undefined);
  let keysFromUser = $state<ProdKeys | undefined>(undefined);

  const setMainKeys = (newKeys: ProdKeys | undefined) => (keysFromMain = newKeys);
  const setUserKeys = (newKeys: ProdKeys | undefined) => (keysFromUser = newKeys);

  return {
    setMainKeys,
    setUserKeys,
    get userKeysSelected() {
      return keysFromUser !== undefined;
    },
    get value() {
      return keysFromUser ?? keysFromMain;
    },
  };
}

export const keys = createKeyStore();
