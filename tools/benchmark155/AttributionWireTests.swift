import Foundation
import Testing
import Tokenizers
import MLXHuggingFace
import MLXLMCommon
import MLXGuidedGeneration
@testable import BoundedGeneration

@Suite("Attribution wire audit without model weights")
struct AttributionWireTests {
    struct Message: Decodable { let role: String; let content: String }
    struct Case: Decodable {
        let name: String; let schema: String; let messages: [Message]
        let legal: [String]; let illegal: [String]
    }
    @Test func replayAttributionAndDirectSchemas() async throws {
        let folder=URL(fileURLWithPath:"/Users/Natsuki/Library/Caches/commiter/mlx-models/8d8e620cfcdc2ce0416f0443adea1f985e052260292f3510f55a9cad165e7aa8")
        let cases=try JSONDecoder().decode([Case].self,from:Data(contentsOf:URL(fileURLWithPath:"/tmp/issue155-wire-input.json")))
        let upstream=try await Tokenizers.AutoTokenizer.from(modelFolder:folder)
        let tokenizer = #adaptHuggingFaceTokenizer(upstream)
        let vocabulary=TokenizerVocabExtractor.extractForGrammar(from:tokenizer)
        let eos=try #require(tokenizer.eosTokenId)
        let grammarTokenizer=try GrammarTokenizer(vocab:vocabulary.vocab,vocabType:vocabulary.vocabType,eosTokenId:Int32(eos))
        let config=try JSONSerialization.jsonObject(with:Data(contentsOf:folder.appendingPathComponent("config.json"))) as! [String:Any]
        let stopIDs=(config["eos_token_id"] as? [Int]) ?? [config["eos_token_id"] as? Int ?? eos]
        var accepted=0;var rejected=0;var zeroPenalties=0;var allFit=true;var traces=[[String:Any]]()
        for item in cases {
            let chat:[[String:any Sendable]]=item.messages.map {["role":$0.role,"content":$0.content]}
            let prompt=try tokenizer.applyChatTemplate(messages:chat,tools:nil,additionalContext:["enable_thinking":false])
            let decoded=tokenizer.decode(tokenIds:prompt)
            #expect(decoded.contains(item.messages[0].content) && decoded.contains(item.messages[1].content))
            let fit=prompt.count+1536<=16384;allFit=allFit && fit;#expect(fit)
            let factory:@Sendable () throws -> GrammarConstraint={try GrammarConstraint(tokenizer:grammarTokenizer,jsonSchema:item.schema,fastForward:false)}
            for (legal,outputs) in [(true,item.legal),(false,item.illegal)] {
                for output in outputs {
                    let state=GrammarSamplingState(constraint:try factory(),vocabSize:grammarTokenizer.vocabSize,eos:eos,whitespace:[],runtimeStopIDs:Set(stopIDs).union([eos]),unknownID:tokenizer.unknownTokenId,alignRuntimeStops:true,factory:factory)
                    let tokens=tokenizer.encode(text:output,addSpecialTokens:false)
                    var allowed=true;var penalties=0
                    for token in tokens+[eos] {
                        let mask=state.maskValues(count:grammarTokenizer.vocabSize)
                        if !mask[token].isFinite {allowed=false;break}
                        if mask[token]<0 {penalties+=1}
                        state.commit(token)
                    }
                    if legal {
                        #expect(allowed && state.succeeded && penalties==0)
                        #expect(tokens.count<=1536)
                        if allowed && state.succeeded {accepted+=1}
                        if penalties==0 {zeroPenalties+=1}
                    } else {
                        #expect(!allowed && !state.succeeded)
                        if !allowed {rejected+=1}
                    }
                    traces.append(["case":item.name,"legal":legal,"output_tokens":tokens.count,"prompt_tokens":prompt.count,"allowed":allowed,"penalized_tokens":penalties])
                }
            }
        }
        let report:[String:Any]=["model_calls":0,"weights_loaded":false,"cases":cases.count,"legal_accepted":accepted,"illegal_rejected":rejected,"zero_penalty_replays":zeroPenalties,"all_prompt_tokens_fit":allFit,"traces":traces]
        try JSONSerialization.data(withJSONObject:report,options:[.prettyPrinted,.sortedKeys]).write(to:URL(fileURLWithPath:"/tmp/issue155-wire-audit.json"))
    }
}
