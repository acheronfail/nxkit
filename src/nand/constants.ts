export interface NxPartition {
  name: string;
  bisKeyId: null | 0 | 1 | 2 | 3;
}

export const NX_PARTITIONS: { [guid: string]: NxPartition } = {
  '98109E25-64E2-4C95-8A77-414916F5BCEB': { name: 'PRODINFO', bisKeyId: 0 },
  'F3056AEC-5449-494C-9F2C-5FDCB75B6E6E': { name: 'PRODINFOF', bisKeyId: 0 },
  '5365DE36-911B-4BB4-8FF9-AA1EBCD73990': { name: 'BCPKG2-1-Normal-Main', bisKeyId: null },
  '8455717B-BD2B-4162-8454-91695218FC38': { name: 'BCPKG2-2-Normal-Sub', bisKeyId: null },
  '8ED6C9A6-9C48-490B-BBEB-001D17A4C0F7': { name: 'BCPKG2-3-SafeMode-Main', bisKeyId: null },
  '5E99751C-56C9-47CC-AA30-B65039888917': { name: 'BCPKG2-4-SafeMode-Sub', bisKeyId: null },
  'C447D9A2-24B7-468A-98C8-595CD077165A': { name: 'BCPKG2-5-Repair-Main', bisKeyId: null },
  '9586E1A1-3AA2-4C90-91B3-2F4A5195B4D2': { name: 'BCPKG2-6-Repair-Sub', bisKeyId: null },
  'A44F9F6B-4ED3-441F-A34A-56AAA136BC6A': { name: 'SAFE', bisKeyId: 1 },
  'ACB0CDF0-4F72-432D-AA0D-5388C733B224': { name: 'SYSTEM', bisKeyId: 2 },
  '2B777F63-E842-47AF-94C4-25A7F18B2280': { name: 'USER', bisKeyId: 3 },
};
