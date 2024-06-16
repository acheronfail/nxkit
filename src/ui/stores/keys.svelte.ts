import type { ProdKeys } from '../../channels';

function createKeyStore() {
  let keysFromMain = $state<ProdKeys | null>(null);
  let keysFromUser = $state<ProdKeys | null>(null);

  const setMainKeys = (newKeys: ProdKeys | null) => (keysFromMain = newKeys);
  const setUserKeys = (newKeys: ProdKeys | null) => (keysFromUser = newKeys);

  return {
    setMainKeys,
    setUserKeys,
    get userKeysSelected() {
      return keysFromUser !== null;
    },
    get value() {
      return keysFromUser ?? keysFromMain;
    },
  };
}

export const keys = createKeyStore();
