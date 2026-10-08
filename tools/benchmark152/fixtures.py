"""Fresh author-defined synthetic requirements, fixed before inference."""
import json

def fresh():
    # Unchanged adapters are deliberately outside selected diffs. Their bodies
    # are real repository evidence, never generated from a partition at run time.
    files = [
        dict(ID='F001',Path='arith/scale.go',Before='package arith\nfunc Scale(n int) int { return n * 2 }\n',After='package arith\nfunc Scale(n int) int { return n * 3 }\n'),
        dict(ID='F002',Path='checks/alpha_test.go',Before='package checks\nimport "fixture/adapters"\nfunc Alpha() bool { return adapters.West(2) == 4 }\n',After='package checks\nimport "fixture/adapters"\nfunc Alpha() bool { return adapters.West(2) == 6 }\n'),
        dict(ID='F003',Path='text/suffix.go',Before='package text\nfunc Suffix(s string) string { return s + ".old" }\n',After='package text\nfunc Suffix(s string) string { return s + ".new" }\n'),
        dict(ID='F004',Path='checks/beta_test.go',Before='package checks\nimport "fixture/adapters"\nfunc Beta() bool { return adapters.East("a") == "a.old" }\n',After='package checks\nimport "fixture/adapters"\nfunc Beta() bool { return adapters.East("a") == "a.new" }\n'),
        dict(ID='F005',Path='limits/clamp.go',Before='package limits\nfunc Clamp(n int) int { if n > 8 { return 8 }; return n }\n',After='package limits\nfunc Clamp(n int) int { if n > 9 { return 9 }; return n }\n'),
        dict(ID='F006',Path='checks/gamma_test.go',Before='package checks\nimport "fixture/adapters"\nfunc Gamma() bool { return adapters.North(10) == 8 }\n',After='package checks\nimport "fixture/adapters"\nfunc Gamma() bool { return adapters.North(10) == 9 }\n'),
    ]
    repo = [dict(ID='R001',Path='adapters/wrappers.go',Content='package adapters\nimport("fixture/arith";"fixture/text";"fixture/limits")\nfunc West(n int) int { return arith.Scale(n) }\nfunc East(s string) string { return text.Suffix(s) }\nfunc North(n int) int { return limits.Clamp(n) }\n')]
    bridge = dict(Name='fresh152-unchanged-adapters-6',Files=files,Repository=repo,
                  History=[[0,2,4],[1,3,5],[0,3],[1,4],[2,5]],Gold=[['F001','F002'],['F003','F004'],['F005','F006']])
    # A common unchanged helper is a structural relation, not a shared purpose.
    independent=[]
    for i,(name,old,new) in enumerate([('Red','r','red'),('Green','g','green'),('Blue','b','blue')]):
        independent.append(dict(ID=f'F{2*i+1:03}',Path=f'labels/{name.lower()}.go',Before=f'package labels\nimport "fixture/common"\nfunc {name}(s string) string {{ return "{old}:" + common.Clean(s) }}\n',After=f'package labels\nimport "fixture/common"\nfunc {name}(s string) string {{ return "{new}:" + common.Clean(s) }}\n'))
        independent.append(dict(ID=f'F{2*i+2:03}',Path=f'labels/{name.lower()}_test.go',Before=f'package labels\nfunc Check{name}() bool {{ return {name}("x") == "{old}:x" }}\n',After=f'package labels\nfunc Check{name}() bool {{ return {name}("x") == "{new}:x" }}\n'))
    guard=dict(Name='fresh152-common-helper-independent-6',Files=independent,
               Repository=[dict(ID='R001',Path='common/clean.go',Content='package common\nfunc Clean(s string) string { return s }\n')],
               History=[[0,2,4],[1,3,5],[0,3],[1,4],[2,5]],Gold=[['F001','F002'],['F003','F004'],['F005','F006']])
    return [bridge,guard]

if __name__ == '__main__': print(json.dumps(fresh(),indent=2))
