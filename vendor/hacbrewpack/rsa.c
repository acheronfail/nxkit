#include <stdio.h>
#include <string.h>
#ifdef NXKIT_DETERMINISTIC_RSA_PSS
#include <stdint.h>
#else
#include "mbedtls/entropy.h"
#include "mbedtls/ctr_drbg.h"
#endif
#include "rsa.h"
#include "rsa_keys.h"
#include "mbedtls/md.h"
#include "mbedtls/rsa.h"
#include "mbedtls/x509.h"

#ifdef NXKIT_DETERMINISTIC_RSA_PSS
typedef struct
{
    unsigned char seed[32];
    uint64_t counter;
} deterministic_rng_ctx_t;

static int deterministic_rng(void *p_rng, unsigned char *output, size_t output_size)
{
    deterministic_rng_ctx_t *ctx = (deterministic_rng_ctx_t *)p_rng;
    unsigned char block[32];
    unsigned char input[40];
    size_t offset = 0;

    memcpy(input, ctx->seed, sizeof(ctx->seed));

    while (offset < output_size)
    {
        for (int i = 0; i < 8; i++)
            input[32 + i] = (unsigned char)((ctx->counter >> (i * 8)) & 0xff);

        mbedtls_md(mbedtls_md_info_from_type(MBEDTLS_MD_SHA256), input, sizeof(input), block);
        ctx->counter++;

        size_t copy_size = output_size - offset;
        if (copy_size > sizeof(block))
            copy_size = sizeof(block);

        memcpy(output + offset, block, copy_size);
        offset += copy_size;
    }

    return 0;
}
#endif

void rsa_sign(void* input, size_t input_size, unsigned char* output, size_t output_size)
{
    unsigned char hash[32];
    unsigned char buf[MBEDTLS_MPI_MAX_SIZE];
#ifndef NXKIT_DETERMINISTIC_RSA_PSS
    const char *pers = "rsa_sign_pss";
#endif
    size_t olen = 0;
#ifdef NXKIT_DETERMINISTIC_RSA_PSS
    deterministic_rng_ctx_t rng_ctx;
#endif

    mbedtls_pk_context pk;
#ifndef NXKIT_DETERMINISTIC_RSA_PSS
    mbedtls_entropy_context entropy;
    mbedtls_ctr_drbg_context ctr_drbg;
#endif

#ifndef NXKIT_DETERMINISTIC_RSA_PSS
    mbedtls_entropy_init(&entropy);
#endif
    mbedtls_pk_init(&pk);
#ifndef NXKIT_DETERMINISTIC_RSA_PSS
    mbedtls_ctr_drbg_init(&ctr_drbg);
#endif

    // Parse private key and sign input
#ifndef NXKIT_DETERMINISTIC_RSA_PSS
    mbedtls_ctr_drbg_seed(&ctr_drbg, mbedtls_entropy_func, &entropy, (const unsigned char *)pers, strlen(pers));
#endif
    mbedtls_pk_parse_key(&pk, (unsigned char*)rsa_private_key, strlen(rsa_private_key) + 1, NULL, 0);
    mbedtls_rsa_set_padding(mbedtls_pk_rsa(pk), MBEDTLS_RSA_PKCS_V21, MBEDTLS_MD_SHA256);
    mbedtls_md(mbedtls_md_info_from_type(MBEDTLS_MD_SHA256), (unsigned char*)input, input_size, hash);

#ifdef NXKIT_DETERMINISTIC_RSA_PSS
    // Reproducible NSPs need a deterministic RSA-PSS salt. This private key is bundled
    // with hacbrewpack, so build reproducibility is more useful than signature entropy.
    memset(&rng_ctx, 0, sizeof(rng_ctx));
    memcpy(rng_ctx.seed, hash, sizeof(hash));
    mbedtls_pk_sign(&pk, MBEDTLS_MD_SHA256, hash, 0, buf, &olen, deterministic_rng, &rng_ctx);
#else
    mbedtls_pk_sign(&pk, MBEDTLS_MD_SHA256, hash, 0, buf, &olen, mbedtls_ctr_drbg_random, &ctr_drbg);
#endif

    // Copy signature to output
    memcpy(output, buf, output_size);

    mbedtls_pk_free(&pk);
#ifndef NXKIT_DETERMINISTIC_RSA_PSS
    mbedtls_ctr_drbg_free(&ctr_drbg);
    mbedtls_entropy_free(&entropy);
#endif
}

const unsigned char *rsa_get_public_key()
{
    return rsa_public_key;
}
