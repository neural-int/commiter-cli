// Compile the actual pinned adapter, not a second implementation of it.
// Tests use the real LLGuidance library and a synthetic single-byte vocabulary.
#include "llguidance.h"
#include "llguidance.cpp"

#include <cstdio>

static int continued() {
    std::fprintf(stderr, "FAIL: LLGuidance failure returned control to generation\n");
    return 90;
}

int main(int argc, char ** argv) {
    if (argc != 2) {
        return 2;
    }
    const std::string mode = argv[1];
    uint32_t lengths[257];
    uint8_t bytes[256];
    for (int i = 0; i < 256; ++i) {
        lengths[i] = 1;
        bytes[i] = i;
    }
    lengths[256] = 0;
    LlgTokenizerInit init = {};
    init.vocab_size = 257;
    init.tok_eos = 256;
    init.token_lens = lengths;
    init.token_bytes = bytes;
    init.use_approximate_greedy_tokenize_fn = true;
    char error[1024] = {};
    auto * tokenizer = llg_new_tokenizer(&init, error, sizeof(error));
    if (!tokenizer) {
        std::fprintf(stderr, "test tokenizer initialization failed\n");
        return 2;
    }
    if (mode == "compile") {
        llama_sampler_llg_new(tokenizer, "lark", "start: (");
        return continued();
    }
    LlgConstraintInit constraint;
    llg_constraint_init_set_defaults(&constraint, tokenizer);
    if (mode == "mask") {
        // Real parser resource exhaustion during mask computation, after a
        // successful grammar compile. No production fault-injection hook.
        constraint.limits.step_max_items = 1;
    }
    const char * grammar = mode == "mask" ? "start: \"[\" start \"]\" | \"a\"" :
        (mode == "normal-terminal" || mode == "terminal-reject" ? "start: \"a\"" : "start: /[a-z]+/");
    auto * matcher = llg_new_matcher(&constraint, "lark", grammar);
    if (!matcher || llg_matcher_get_error(matcher)) {
        std::fprintf(stderr, "test grammar failed before the intended operation\n");
        return 2;
    }
    llama_sampler_llg state = {nullptr, "lark", grammar, tokenizer, matcher};
    llama_sampler sampler = {&llama_sampler_llg_i, &state};
    llama_token_data candidate = {'a', 0.0f, 0.0f};
    llama_token_data_array candidates = {&candidate, 1, -1, false};
    if (mode == "mask") {
        auto * probe = llg_clone_matcher(matcher);
        const int status = llg_matcher_compute_mask(probe);
        llg_free_matcher(probe);
        if (status == 0) {
            std::fprintf(stderr, "mask fixture did not exhaust parser resources\n");
            return 2;
        }
        std::fprintf(stderr, "real mask failure reproduced: status=%d\n", status);
        llama_sampler_llg_apply(&sampler, &candidates);
        return continued();
    }
    if (mode == "consume") {
        llama_sampler_llg_accept_impl(&sampler, '!');
        return continued();
    }
    if (mode == "reset" || mode == "clone" || mode == "sticky") {
        if (llg_matcher_consume_token(matcher, '!') == 0 || !llg_matcher_get_error(matcher)) {
            return 2;
        }
        if (mode == "reset") {
            llama_sampler_llg_reset(&sampler);
        } else if (mode == "clone") {
            llama_sampler_llg_clone(&sampler);
        } else {
            llama_sampler_llg_apply(&sampler, &candidates);
        }
        return continued();
    }
    if (mode == "terminal-reject") {
        llama_sampler_llg_accept_impl(&sampler, 'a');
        llama_sampler_llg_accept_impl(&sampler, 'b');
        return continued();
    }
    if (mode != "normal" && mode != "normal-terminal") {
        return 2;
    }
    llama_sampler_llg_apply(&sampler, &candidates);
    if (!std::isfinite(candidate.logit)) {
        return 2;
    }
    llama_sampler_llg_accept_impl(&sampler, 'a');
    candidate = {256, 0.0f, 0.0f};
    llama_sampler_llg_apply(&sampler, &candidates);
    if (!std::isfinite(candidate.logit)) {
        return 2;
    }
    llama_sampler_llg_accept_impl(&sampler, 256);
    llama_sampler_llg_reset(&sampler);
    auto * clone = llama_sampler_llg_clone(&sampler);
    llama_sampler_free(clone);
    llg_free_matcher(matcher);
    llg_free_tokenizer(tokenizer);
    std::fprintf(stderr, "PASS: constrained token, EOS, reset and clone\n");
    return 0;
}
