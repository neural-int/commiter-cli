import argparse,pathlib,json,subprocess,importlib.util
root=pathlib.Path(__file__).resolve().parents[2]
parser=argparse.ArgumentParser(description='Build the explicit no-deadline, no-file-count history/inference study overlay; no model calls')
parser.add_argument('--out',type=pathlib.Path,required=True)
parser.add_argument('--fixed-context-32k',action='store_true')
parser.add_argument('--output-limit',type=int,choices=[4096])
args=parser.parse_args()
out=args.out.resolve();out.mkdir(mode=0o700,exist_ok=False)
build=out/'build';build.mkdir()
spec=importlib.util.spec_from_file_location('build',root/'tools/benchmark182/build.py');module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
replacements={}
def replace(relative,old,new):
 source=root/relative;text=source.read_text();assert old in text,relative
 target=build/(source.parent.name+'-'+source.name);target.write_text(text.replace(old,new));replacements[str(source)]=str(target)
source=root/'internal/planning/three_phase.go';target=build/'three_phase.go';text=module.planner_overlay(source.read_text(),True).replace('context.WithTimeout(parent, CandidateCycleTimeout)','context.WithCancel(parent)');target.write_text(text);replacements[str(source)]=str(target)
replace('internal/cli/attributed_history.go','context.WithTimeout(ctx, 5*time.Second)','context.WithCancel(ctx)')
replace('internal/historyattribution/pipeline.go','context.WithTimeout(parent, 5*time.Second)','context.WithCancel(parent)')
source=root/'internal/historyattribution/history_timing.go';target=build/'history_timing.go';target.write_text(source.read_text()+'\nfunc StudyTimingContext(ctx context.Context, t *HistoryTiming) context.Context { return context.WithValue(ctx, pipelineWaitContext{}, true) }\n');replacements[str(source)]=str(target)
source=root/'internal/historyattribution/pipeline.go';target=pathlib.Path(replacements[str(source)]);text=target.read_text();text=text.replace('h = readHistory(ctx,','historyReadStart := time.Now()\n            h = readHistory(ctx,').replace('\n\t\t}\n\t}\n\tsources :=','\n            if p:=timing(ctx); p!=nil {p.TotalMS = elapsedMS(historyReadStart)}\n\t\t}\n\t}\n\tsources :=');text=text.replace('type Result struct {','type Result struct {\n StudyPerformance *PipelineWaits\n ContentMatchMS float64\n CommandProjectionMS float64\n HistoryReadMS float64')
text=text.replace('attributed, err := attributeWithSources(request, h, sources)','matchStart:=time.Now()\n attributed, err := attributeWithSources(request, h, sources)\n matchMS:=elapsedMS(matchStart)')
text=text.replace('appendCommandContexts(request, h, attributed, sources, commandSources)','commandStart:=time.Now()\n appendCommandContexts(request, h, attributed, sources, commandSources)\n commandMS:=elapsedMS(commandStart)')
text=text.replace('return Result{ContextSources:', 'return Result{StudyPerformance:h.Waits, ContentMatchMS:matchMS, CommandProjectionMS:commandMS, HistoryReadMS:h.ElapsedMS, ContextSources:')
target.write_text(text)
source=root/'internal/cli/attributed_history_live_test.go' ;target=build/'inference_e2e_test.go';target.write_text((root/'tools/benchmark182/testdata/inference_e2e_test.go.txt').read_text());replacements[str(source)]=str(target)
# Add timing per phase around host payload projection, call, strict JSON decoding.
source=root/'internal/planning/three_phase.go';target=pathlib.Path(replacements[str(source)]);text=target.read_text();text=text.replace('"fmt"','"fmt"\n"os"\n"path/filepath"')
text=text.replace('invoke := func(system string, payload any, schema any, profile string, budget int, output any) error {','''invoke := func(system string, payload any, schema any, profile string, budget int, output any) error {
        stageStart:=time.Now(); stageTimes:=map[string]float64{}
        defer func(){stageTimes["total_seconds"]=time.Since(stageStart).Seconds();if out:=os.Getenv("COMMITER_STUDY_OUT");out!="" {b,_:=json.Marshal(stageTimes);os.WriteFile(filepath.Join(out,profile+"-host-times.json"),b,0600)}}()
        stageCheckpoint:=time.Now()''')
text=text.replace('encoded, e = budgetPhaseHistory(', 'stageTimes["serialize_and_select_tier_seconds"]=time.Since(stageCheckpoint).Seconds();stageCheckpoint=time.Now()\n        encoded, e = budgetPhaseHistory(')
text=text.replace('result.Calls++','stageTimes["history_projection_and_budget_seconds"]=time.Since(stageCheckpoint).Seconds();stageCheckpoint=time.Now()\n        result.Calls++')
text=text.replace('if e != nil {\n\t\t\treturn generationError(nil)','stageTimes["client_call_seconds"]=time.Since(stageCheckpoint).Seconds();stageCheckpoint=time.Now()\n        if e != nil {\n\t\t\treturn generationError(nil)')
text=text.replace('if e := strictCandidateJSON(candidate, output); e != nil {','defer func(){stageTimes["response_safety_and_json_validation_seconds"]=time.Since(stageCheckpoint).Seconds()}()\n        if e := strictCandidateJSON(candidate, output); e != nil {');target.write_text(text)
if args.fixed_context_32k:
 source=root/'internal/planning/three_phase.go';target=pathlib.Path(replacements[str(source)]);text=target.read_text()
 start=text.index('\t\tcontextTokens := 0');end=text.index('\t\t// Optional evidence',start)
 text=text[:start]+'\t\tcontextTokens := contextinput.Context32K\n\t\tif requiredTokens > contextTokens { return contextinput.ErrTooLarge }\n'+text[end:];target.write_text(text)
 source=root/'internal/cli/attributed_history_live_test.go';target=pathlib.Path(replacements[str(source)]);text=target.read_text().replace('window:=min(o.ContextTokens,c.maxContext)', 'window:=o.ContextTokens;if window!=32768 || c.maxContext<32768 {return llm.Response{},errors.New("fixed 32k contract violated")}')
 text=text.replace('fixture.Prepared.Prompt=fixture.Prompt','fixture.Prepared.Prompt=fixture.Prompt;fixture.Prepared.Budget.ContextTokens=32768');target.write_text(text)
 if args.output_limit:
  text=target.read_text().replace('OutputTokens:0','OutputTokens:4096');target.write_text(text)
 (build/'context-policy.json').write_text(json.dumps({'context_tokens':32768,'automatic_selection':False,'output_limit':args.output_limit}))
(build/'overlay.json').write_text(json.dumps({'Replace':replacements},indent=2))
subprocess.run(['go','test','-overlay='+str(build/'overlay.json'),'-c','-o',str(build/'evaluation.test'),'./internal/cli'],cwd=root,check=True)
