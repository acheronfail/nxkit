#include <math.h>
#include <node.h>
#include <node_buffer.h>
#include <openssl/evp.h>

namespace xtsn {

using v8::Context;
using v8::External;
using v8::Function;
using v8::FunctionCallbackInfo;
using v8::FunctionTemplate;
using v8::Isolate;
using v8::Local;
using v8::NewStringType;
using v8::Number;
using v8::Object;
using v8::Persistent;
using v8::String;
using v8::Value;

void CreateXtsnCipherInstance(const FunctionCallbackInfo<Value> &args) {
  Isolate *isolate = args.GetIsolate();

  if (!args.IsConstructCall()) {
    isolate->ThrowException(String::NewFromUtf8(isolate, "Constructor XtsnCipher requires 'new'").ToLocalChecked());
    return;
  }

  if (args.Length() < 3 || !node::Buffer::HasInstance(args[0]) || !node::Buffer::HasInstance(args[1]) ||
      !args[2]->IsNumber()) {
    isolate->ThrowException(String::NewFromUtf8(isolate, "invalid arguments").ToLocalChecked());
    return;
  }

  const unsigned char *cryptoKeyData = reinterpret_cast<unsigned char *>(node::Buffer::Data(args[0]));
  if (node::Buffer::Length(args[0]) != 16) {
    isolate->ThrowException(String::NewFromUtf8(isolate, "crypto key must be 16 bytes exactly").ToLocalChecked());
    return;
  }

  const unsigned char *tweakKeyData = reinterpret_cast<unsigned char *>(node::Buffer::Data(args[1]));
  if (node::Buffer::Length(args[1]) != 16) {
    isolate->ThrowException(String::NewFromUtf8(isolate, "tweak key must be 16 bytes exactly").ToLocalChecked());
    return;
  }

  EVP_CIPHER_CTX *ctx_tweak = EVP_CIPHER_CTX_new();
  EVP_CIPHER_CTX *ctx_crypto_encrypt = EVP_CIPHER_CTX_new();
  EVP_CIPHER_CTX *ctx_crypto_decrypt = EVP_CIPHER_CTX_new();

  assert(EVP_EncryptInit(ctx_tweak, EVP_aes_128_ecb(), tweakKeyData, nullptr));
  assert(EVP_EncryptInit(ctx_crypto_encrypt, EVP_aes_128_ecb(), cryptoKeyData, nullptr));
  assert(EVP_DecryptInit(ctx_crypto_decrypt, EVP_aes_128_ecb(), cryptoKeyData, nullptr));

  assert(EVP_CIPHER_CTX_set_padding(ctx_tweak, 0));
  assert(EVP_CIPHER_CTX_set_padding(ctx_crypto_encrypt, 0));
  assert(EVP_CIPHER_CTX_set_padding(ctx_crypto_decrypt, 0));

  args.This()->SetAlignedPointerInInternalField(0, ctx_tweak);
  args.This()->SetAlignedPointerInInternalField(1, ctx_crypto_encrypt);
  args.This()->SetAlignedPointerInInternalField(2, ctx_crypto_decrypt);
  args.This()->SetInternalField(3, args[2]);

  args.GetReturnValue().Set(args.This());
}

void InitTweak(EVP_CIPHER_CTX *ctx_tweak, unsigned char *tweak, uint64_t sectorOffset) {
#if defined(__GNUC__) || defined(__clang__)
  uint64_t sectorOffsetBigEndian = __builtin_bswap64(sectorOffset);
  memcpy(tweak + 8, &sectorOffsetBigEndian, sizeof(uint64_t));
#elif defined(_MSC_VER)
  uint64_t sectorOffsetBigEndian = _byteswap_uint64(sectorOffset);
  memcpy(tweak + 8, &sectorOffsetBigEndian, sizeof(uint64_t));
#else
  for (int i = 0; i < sizeof(uint64_t); i++)
    tweak[15 - i] = ((unsigned char *)&sectorOffset)[i];
#endif

  int outputLen = 0;
  assert(EVP_CipherUpdate(ctx_tweak, tweak, &outputLen, tweak, 16));
}

inline void UpdateTweak(unsigned char *tweak) {
  bool last_high = (bool)(tweak[15] & 0x80);
  for (int j = 15; j > 0; j--)
    tweak[j] = (unsigned char)(((tweak[j] << 1) & ~1) | (tweak[j - 1] & 0x80 ? 1 : 0));

  tweak[0] = (unsigned char)(((tweak[0] << 1) & ~1) ^ (last_high ? 0x87 : 0));
}

inline void ProcessChunks(EVP_CIPHER_CTX *ctx_crypto, unsigned char *input, unsigned char *tweak, uint64_t *chunkOffset,
                         uint64_t totalChunks, int runs) {
  uint64_t *tweak64bit = reinterpret_cast<uint64_t *>(tweak);
  uint64_t *input64bit = reinterpret_cast<uint64_t *>(input);

  int unusedOutputLen = 0;
  for (int i = 0; i < runs; i++) {
    if (*chunkOffset >= totalChunks)
      return;

    input64bit[*chunkOffset + 0] ^= tweak64bit[0];
    input64bit[*chunkOffset + 1] ^= tweak64bit[1];

    unsigned char *block = input + (*chunkOffset * 8);
    assert(EVP_CipherUpdate(ctx_crypto, block, &unusedOutputLen, block, 16));

    input64bit[*chunkOffset + 0] ^= tweak64bit[0];
    input64bit[*chunkOffset + 1] ^= tweak64bit[1];

    UpdateTweak(tweak);

    // we're processing two 64bit chunks at a time (the cipher is a 128bit cipher)
    *chunkOffset += 2;
  }
}

void RunCipherMethod(const FunctionCallbackInfo<Value> &args) {
  Isolate *isolate = args.GetIsolate();

  // validate arguments from js
  if (args.Length() < 4 || !node::Buffer::HasInstance(args[0]) || !args[1]->IsNumber() || !args[2]->IsNumber()) {
    isolate->ThrowException(String::NewFromUtf8(isolate, "invalid arguments").ToLocalChecked());
    return;
  }

  // extract arguments from js
  unsigned char *input = reinterpret_cast<unsigned char *>(node::Buffer::Data(args[0]));
  uint64_t inputLen = node::Buffer::Length(args[0]);
  uint64_t sectorOffset = args[1]->NumberValue(isolate->GetCurrentContext()).FromJust();
  uint64_t skippedBytes = args[2]->NumberValue(isolate->GetCurrentContext()).FromJust();
  bool encrypt = args[3]->BooleanValue(isolate);

  // extract saved fields
  int sectorSize = args.This()->GetInternalField(3).As<Number>()->NumberValue(isolate->GetCurrentContext()).FromJust();
  EVP_CIPHER_CTX *ctx_tweak = reinterpret_cast<EVP_CIPHER_CTX *>(args.Holder()->GetAlignedPointerFromInternalField(0));
  EVP_CIPHER_CTX *ctx_crypto =
      reinterpret_cast<EVP_CIPHER_CTX *>(args.Holder()->GetAlignedPointerFromInternalField(encrypt ? 1 : 2));

  // we measure our current offset through the input in 64bit chunks
  uint64_t chunkOffset = 0;
  uint64_t totalChunks = inputLen / 8;

  // see if we can completely skip any sectors
  if (skippedBytes > 0) {
    int fullSectorsToSkip = std::floor(skippedBytes / sectorSize);
    sectorOffset += fullSectorsToSkip;
    skippedBytes %= sectorSize;
  }

  // if we have remaining skipped bytes, that means we're up to the first sector
  // we need to run the cipher on
  if (skippedBytes > 0) {
    unsigned char tweak[16] = {0};
    InitTweak(ctx_tweak, tweak, sectorOffset);

    // update the tweak for all chunks before the first chunk we need to process
    for (int i = 0; i < std::floor(skippedBytes / 16); i++) {
      UpdateTweak(tweak);
    }

    // finally, process the rest of the chunks in this sector
    ProcessChunks(ctx_crypto, input, tweak, &chunkOffset, totalChunks, std::floor((sectorSize - skippedBytes) / 16));
    sectorOffset++;
  }

  // now we're sector-aligned and can proceed to process each remaining sector
  while (chunkOffset < totalChunks) {
    unsigned char tweak[16] = {0};
    InitTweak(ctx_tweak, tweak, sectorOffset);
    ProcessChunks(ctx_crypto, input, tweak, &chunkOffset, totalChunks, std::floor(sectorSize / 16));
    sectorOffset++;
  }
}

void Initialize(Local<Object> exports, Local<Object> module) {
  Isolate *isolate = exports->GetIsolate();

  // setup storage for the XtsnCipher class
  Local<FunctionTemplate> tpl = FunctionTemplate::New(isolate, CreateXtsnCipherInstance);
  tpl->SetClassName(String::NewFromUtf8(isolate, "XtsnCipher").ToLocalChecked());
  tpl->InstanceTemplate()->SetInternalFieldCount(4);

  // setup methods for XtsnCipher instances
  NODE_SET_PROTOTYPE_METHOD(tpl, "run", RunCipherMethod);

  // set the default export to be the XtsnCipher constructor
  auto constructorFunction = tpl->GetFunction(isolate->GetCurrentContext()).ToLocalChecked();
  module
      ->Set(isolate->GetCurrentContext(), String::NewFromUtf8(isolate, "exports").ToLocalChecked(), constructorFunction)
      .FromJust();
}

NODE_MODULE(NODE_GYP_MODULE_NAME, Initialize)

} // namespace xtsn