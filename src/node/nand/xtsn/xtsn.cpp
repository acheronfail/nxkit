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
using v8::WeakCallbackInfo;
using v8::WeakCallbackType;

class Ciphers {
public:
  /**
   * 0: tweak
   * 1: crypto encrypt
   * 2: crypto decrypt
   */
  EVP_CIPHER_CTX *ctx_list[3];

  /**
   * Required to be able to run code after the JS object is GC'd
   */
  Persistent<Object> persistent;

  Ciphers(Isolate *isolate, Local<Object> jsObj, unsigned char *tweakKeyData, unsigned char *cryptoKeyData)
      : persistent(isolate, jsObj) {
    ctx_list[0] = EVP_CIPHER_CTX_new();
    ctx_list[1] = EVP_CIPHER_CTX_new();
    ctx_list[2] = EVP_CIPHER_CTX_new();

    assert(EVP_EncryptInit(ctx_list[0], EVP_aes_128_ecb(), tweakKeyData, nullptr));
    assert(EVP_EncryptInit(ctx_list[1], EVP_aes_128_ecb(), cryptoKeyData, nullptr));
    assert(EVP_DecryptInit(ctx_list[2], EVP_aes_128_ecb(), cryptoKeyData, nullptr));

    assert(EVP_CIPHER_CTX_set_padding(ctx_list[0], 0));
    assert(EVP_CIPHER_CTX_set_padding(ctx_list[1], 0));
    assert(EVP_CIPHER_CTX_set_padding(ctx_list[2], 0));

    // setup callback to run after the js object is GC'd
    this->persistent.SetWeak(this, Ciphers::FreeXtsnCipherInstance, WeakCallbackType::kParameter);
  }

  ~Ciphers() {
    EVP_CIPHER_CTX_free(ctx_list[0]);
    EVP_CIPHER_CTX_free(ctx_list[1]);
    EVP_CIPHER_CTX_free(ctx_list[2]);
  }

  static void FreeXtsnCipherInstance(const WeakCallbackInfo<Ciphers> &info) {
    Ciphers *ciphers = reinterpret_cast<Ciphers *>(info.GetParameter());

    // V8 requires that we reset this when the weak callback is run
    ciphers->persistent.Reset();

    // this calls the destructor and cleans up
    delete ciphers;
  }
};

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

  unsigned char *cryptoKeyData = reinterpret_cast<unsigned char *>(node::Buffer::Data(args[0]));
  if (node::Buffer::Length(args[0]) != 16) {
    isolate->ThrowException(String::NewFromUtf8(isolate, "crypto key must be 16 bytes exactly").ToLocalChecked());
    return;
  }

  unsigned char *tweakKeyData = reinterpret_cast<unsigned char *>(node::Buffer::Data(args[1]));
  if (node::Buffer::Length(args[1]) != 16) {
    isolate->ThrowException(String::NewFromUtf8(isolate, "tweak key must be 16 bytes exactly").ToLocalChecked());
    return;
  }

  Local<Object> jsObject = args.This();
  // create our ciphers class and attach it to the JS object we're creating now
  Ciphers *ciphers = new Ciphers(isolate, jsObject, tweakKeyData, cryptoKeyData);
  // store ciphers on js object
  jsObject->SetAlignedPointerInInternalField(0, ciphers);
  // store sector size on js object
  jsObject->SetInternalField(1, args[2]);

  args.GetReturnValue().Set(jsObject);
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

  uint64_t *tweak64bit = reinterpret_cast<uint64_t *>(tweak);
  tweak64bit[1] = tweak64bit[1] << 1 | (tweak64bit[0] >> 63);
  tweak64bit[0] = tweak64bit[0] << 1 ^ (last_high * 0x87);
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
  int sectorSize = args.This()->GetInternalField(1).As<Number>()->NumberValue(isolate->GetCurrentContext()).FromJust();

  Ciphers *ciphers = reinterpret_cast<Ciphers *>(args.Holder()->GetAlignedPointerFromInternalField(0));
  EVP_CIPHER_CTX *ctx_tweak = ciphers->ctx_list[0];
  EVP_CIPHER_CTX *ctx_crypto = ciphers->ctx_list[encrypt ? 1 : 2];

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
  tpl->InstanceTemplate()->SetInternalFieldCount(2);

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