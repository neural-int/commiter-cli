"""Official mistral-common input serialization for the opt-in Swift benchmark.

Reads one JSON request on stdin. No network access; assets are predownloaded and
revision/hash-pinned in the run contract. Output decoding remains in Swift after
a complete byte-vocabulary audit against the official tokenizer.
"""
import argparse
import json
import sys
from pathlib import Path

from mistral_common.protocol.instruct.request import ChatCompletionRequest
from mistral_common.tokens.tokenizers.mistral import MistralTokenizer


def tokenize(request, assets, reasoning):
    tokenizer = MistralTokenizer.from_file(str(assets / 'tekken.json'))
    messages = [dict(message) for message in request['messages']]
    if reasoning:
        official = (assets / 'SYSTEM_PROMPT.txt').read_text()
        # Preserve official reasoning guidance; the benchmark's final JSON
        # contract takes precedence over its generic Markdown presentation.
        messages[0]['content'] += '\n\n' + official + '\n\nFor this task, the final response must be only the required JSON, without Markdown.'
    encoded = tokenizer.encode_chat_completion(ChatCompletionRequest(messages=messages))
    base = tokenizer.instruct_tokenizer.tokenizer
    return {
        'tokens': encoded.tokens,
        'eos': base.eos_id,
        'open': base.get_control_token('[THINK]') if reasoning else None,
        'close': base.get_control_token('[/THINK]') if reasoning else None,
    }


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--assets', type=Path, required=True)
    parser.add_argument('--reasoning', action='store_true')
    args = parser.parse_args()
    print(json.dumps(tokenize(json.load(sys.stdin), args.assets, args.reasoning)))
