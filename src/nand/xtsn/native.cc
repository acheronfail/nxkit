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

void CreateThingInstance(const FunctionCallbackInfo<Value> &args) {
  Isolate *isolate = args.GetIsolate();

  if (!args.IsConstructCall()) {
    isolate->ThrowException(String::NewFromUtf8(isolate, "Constructor Cipher requires 'new'").ToLocalChecked());
    return;
  }

  if (args.Length() < 2 || !node::Buffer::HasInstance(args[0])) {
    isolate->ThrowException(String::NewFromUtf8(isolate, "invalid arguments").ToLocalChecked());
    return;
  }

  // Convert the first argument to a Buffer and extract the data.
  const unsigned char *keyData = reinterpret_cast<unsigned char *>(node::Buffer::Data(args[0]));
  if (node::Buffer::Length(args[0]) != 16) {
    isolate->ThrowException(String::NewFromUtf8(isolate, "key must be 16 bytes exactly").ToLocalChecked());
    return;
  }

  bool encrypt = args[1]->BooleanValue(isolate);

  EVP_CIPHER_CTX *ctx = EVP_CIPHER_CTX_new();
  if (encrypt) {
    EVP_EncryptInit(ctx, EVP_aes_128_ecb(), keyData, nullptr);
  } else {
    EVP_DecryptInit(ctx, EVP_aes_128_ecb(), keyData, nullptr);
  }
  EVP_CIPHER_CTX_set_padding(ctx, 0);

  args.This()->SetAlignedPointerInInternalField(0, ctx);
  args.GetReturnValue().Set(args.This());
}

void UpdateMethod(const FunctionCallbackInfo<Value> &args) {
  Isolate *isolate = args.GetIsolate();

  EVP_CIPHER_CTX *ctx = reinterpret_cast<EVP_CIPHER_CTX *>(args.Holder()->GetAlignedPointerFromInternalField(0));
  unsigned char *data = reinterpret_cast<unsigned char *>(node::Buffer::Data(args[0]));
  int offset = args[1]->IsUndefined() ? 0 : args[1]->NumberValue(isolate->GetCurrentContext()).FromJust();

  int outputLen = 0;
  if (!EVP_CipherUpdate(ctx, data + offset, &outputLen, data + offset, 16)) {
    isolate->ThrowException(String::NewFromUtf8(isolate, "encryption failed").ToLocalChecked());
    return;
  }

  args.GetReturnValue().Set(Number::New(isolate, outputLen));
}

void Initialize(Local<Object> exports, Local<Object> module) {
  Isolate *isolate = exports->GetIsolate();

  Local<FunctionTemplate> tpl = FunctionTemplate::New(isolate, CreateThingInstance);
  tpl->SetClassName(String::NewFromUtf8(isolate, "XtsnCipher").ToLocalChecked());

  // allocate 1 internal field to stort the cipher ctx
  tpl->InstanceTemplate()->SetInternalFieldCount(1);

  NODE_SET_PROTOTYPE_METHOD(tpl, "update", UpdateMethod);
  auto constructorFunction = tpl->GetFunction(isolate->GetCurrentContext()).ToLocalChecked();
  module
      ->Set(isolate->GetCurrentContext(), String::NewFromUtf8(isolate, "exports").ToLocalChecked(), constructorFunction)
      .FromJust();
}

NODE_MODULE(NODE_GYP_MODULE_NAME, Initialize)

} // namespace xtsn