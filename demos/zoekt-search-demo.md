# Zoekt Code Search Skill Demo

*2026-04-13T16:13:20Z by Showboat 0.6.1*
<!-- showboat-id: c1f20321-d723-4104-b19f-b9d4da7e7143 -->

This demo exercises the zoekt-search skill — a fast trigram-indexed code search powered by Zoekt (the engine behind GitHub and Sourcegraph code search). We demonstrate installation, indexing, searching, and the integration test.

## Step 1: Install (first-time setup is automatic)

Zoekt binaries are auto-installed on first use. The install script builds from source using `go install`. Let's verify the binaries are in place.

```bash
ls -la .agents/skills/zoekt-search/bin/
```

```output
total 126048
drwxr-xr-x@ 5 jack  staff       160 13 Apr 17:05 .
drwxr-xr-x@ 5 jack  staff       160 13 Apr 17:10 ..
-rw-r--r--@ 1 jack  staff         0 13 Apr 17:04 .gitkeep
-rwxr-xr-x@ 1 jack  staff  38081202 13 Apr 17:05 zoekt
-rwxr-xr-x@ 1 jack  staff  26451154 13 Apr 17:05 zoekt-index
```

```bash
bash .agents/skills/zoekt-search/scripts/install-zoekt.sh
```

```output
[install-zoekt] Starting zoekt installation (version=latest)
[install-zoekt] Detected platform: darwin-arm64
[install-zoekt] zoekt binaries already installed and working in /Users/jack/Code/withakay/kocao/ito-worktrees/008-01_zoekt-search-skill/.agents/skills/zoekt-search/scripts/../bin
[install-zoekt] Version pinning not requested; skipping re-install.
```

## Step 2: Index the Kocao repo

Index the current worktree. The index is stored in the git directory and is automatically gitignored.

```bash
bash .agents/skills/zoekt-search/scripts/zoekt-index.sh 2>&1
```

```output
[zoekt-index] Using binary: /Users/jack/Code/withakay/kocao/ito-worktrees/008-01_zoekt-search-skill/.agents/skills/zoekt-search/scripts/../bin/zoekt-index
[zoekt-index] Indexing: /Users/jack/Code/withakay/kocao/ito-worktrees/008-01_zoekt-search-skill
[zoekt-index] Index dir: /Users/jack/Code/withakay/kocao/.bare/worktrees/008-01_zoekt-search-skill/zoekt
2026/04/13 17:13:55 finished shard /Users/jack/Code/withakay/kocao/.bare/worktrees/008-01_zoekt-search-skill/zoekt/008-01_zoekt-search-skill_v16.00000.zoekt: 6529579 index bytes (overhead 3.1), 722 files processed 
[zoekt-index] Indexed /Users/jack/Code/withakay/kocao/ito-worktrees/008-01_zoekt-search-skill → /Users/jack/Code/withakay/kocao/.bare/worktrees/008-01_zoekt-search-skill/zoekt
[zoekt-index]   Shards: 1
[zoekt-index]   Index size: 6.2M
```

## Step 3: Search for "AgentSession" — JSONL results

Default output is JSONL with base64-encoded line content. FileName and LineNumber fields are plain text.

```bash
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh 'AgentSession' 2>/dev/null | head -5
```

```output
{"FileName":"web/src/ui/lib/api.ts","Repository":"008-01_zoekt-search-skill","Language":"TypeScript","LineMatches":[{"Line":"ZXhwb3J0IHR5cGUgQWdlbnRTZXNzaW9uSW5mbyA9IHsK","LineStart":129,"LineEnd":162,"LineNumber":9,"Before":null,"After":null,"FileName":false,"Score":501,"DebugScore":"","LineFragments":[{"LineOffset":12,"Offset":141,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICBhZ2VudFNlc3Npb24/OiBBZ2VudFNlc3Npb25JbmZvCg==","LineStart":426,"LineEnd":460,"LineNumber":25,"Before":null,"After":null,"FileName":false,"Score":500.92857142857144,"DebugScore":"","LineFragments":[{"LineOffset":17,"Offset":443,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ZXhwb3J0IHR5cGUgQWdlbnRTZXNzaW9uU3RhdGUgPSB7Cg==","LineStart":542,"LineEnd":576,"LineNumber":31,"Before":null,"After":null,"FileName":false,"Score":500.85714285714283,"DebugScore":"","LineFragments":[{"LineOffset":12,"Offset":554,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ZXhwb3J0IHR5cGUgQWdlbnRTZXNzaW9uRXZlbnQgPSB7Cg==","LineStart":739,"LineEnd":773,"LineNumber":42,"Before":null,"After":null,"FileName":false,"Score":500.7857142857143,"DebugScore":"","LineFragments":[{"LineOffset":12,"Offset":751,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICBnZXRBZ2VudFNlc3Npb246ICh0b2tlbjogc3RyaW5nLCBoYXJuZXNzUnVuSUQ6IHN0cmluZykgPT4K","LineStart":8577,"LineEnd":8637,"LineNumber":361,"Before":null,"After":null,"FileName":false,"Score":500.7142857142857,"DebugScore":"","LineFragments":[{"LineOffset":5,"Offset":8582,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICAgIGFwaUZldGNoPEFnZW50U2Vzc2lvblN0YXRlPihgL2FwaS92MS9oYXJuZXNzLXJ1bnMvJHtlbmNvZGVVUklDb21wb25lbnQoaGFybmVzc1J1bklEKX0vYWdlbnQtc2Vzc2lvbmAsIHsgdG9rZW4gfSksCg==","LineStart":8637,"LineEnd":8755,"LineNumber":362,"Before":null,"After":null,"FileName":false,"Score":500.64285714285717,"DebugScore":"","LineFragments":[{"LineOffset":13,"Offset":8650,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICBjcmVhdGVBZ2VudFNlc3Npb246ICh0b2tlbjogc3RyaW5nLCBoYXJuZXNzUnVuSUQ6IHN0cmluZykgPT4K","LineStart":8755,"LineEnd":8818,"LineNumber":363,"Before":null,"After":null,"FileName":false,"Score":500.57142857142856,"DebugScore":"","LineFragments":[{"LineOffset":8,"Offset":8763,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICAgIGFwaUZldGNoPEFnZW50U2Vzc2lvblN0YXRlPihgL2FwaS92MS9oYXJuZXNzLXJ1bnMvJHtlbmNvZGVVUklDb21wb25lbnQoaGFybmVzc1J1bklEKX0vYWdlbnQtc2Vzc2lvbmAsIHsgbWV0aG9kOiAnUE9TVCcsIHRva2VuIH0pLAo=","LineStart":8818,"LineEnd":8952,"LineNumber":364,"Before":null,"After":null,"FileName":false,"Score":500.5,"DebugScore":"","LineFragments":[{"LineOffset":13,"Offset":8831,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICBwcm9tcHRBZ2VudFNlc3Npb246ICh0b2tlbjogc3RyaW5nLCBoYXJuZXNzUnVuSUQ6IHN0cmluZywgcHJvbXB0OiBzdHJpbmcpID0+Cg==","LineStart":8952,"LineEnd":9031,"LineNumber":365,"Before":null,"After":null,"FileName":false,"Score":500.42857142857144,"DebugScore":"","LineFragments":[{"LineOffset":8,"Offset":8960,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICAgIGFwaUZldGNoPHsgc2Vzc2lvbjogQWdlbnRTZXNzaW9uU3RhdGU7IHJlc3VsdDogdW5rbm93biB9PihgL2FwaS92MS9oYXJuZXNzLXJ1bnMvJHtlbmNvZGVVUklDb21wb25lbnQoaGFybmVzc1J1bklEKX0vYWdlbnQtc2Vzc2lvbi9wcm9tcHRgLCB7Cg==","LineStart":9031,"LineEnd":9176,"LineNumber":366,"Before":null,"After":null,"FileName":false,"Score":500.35714285714283,"DebugScore":"","LineFragments":[{"LineOffset":24,"Offset":9055,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICBsaXN0QWdlbnRTZXNzaW9uRXZlbnRzOiAodG9rZW46IHN0cmluZywgaGFybmVzc1J1bklEOiBzdHJpbmcsIG9wdHM/OiB7IG9mZnNldD86IG51bWJlcjsgbGltaXQ/OiBudW1iZXIgfSkgPT4gewo=","LineStart":9242,"LineEnd":9355,"LineNumber":371,"Before":null,"After":null,"FileName":false,"Score":500.2857142857143,"DebugScore":"","LineFragments":[{"LineOffset":6,"Offset":9248,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICAgIHJldHVybiBhcGlGZXRjaDx7IGV2ZW50czogQWdlbnRTZXNzaW9uRXZlbnRbXTsgbmV4dE9mZnNldDogbnVtYmVyIH0+KGAvYXBpL3YxL2hhcm5lc3MtcnVucy8ke2VuY29kZVVSSUNvbXBvbmVudChoYXJuZXNzUnVuSUQpfS9hZ2VudC1zZXNzaW9uL2V2ZW50cyR7c3VmZml4fWAsIHsgdG9rZW4gfSkK","LineStart":9593,"LineEnd":9767,"LineNumber":376,"Before":null,"After":null,"FileName":false,"Score":500.2142857142857,"DebugScore":"","LineFragments":[{"LineOffset":30,"Offset":9623,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICBzdG9wQWdlbnRTZXNzaW9uOiAodG9rZW46IHN0cmluZywgaGFybmVzc1J1bklEOiBzdHJpbmcpID0+Cg==","LineStart":9772,"LineEnd":9833,"LineNumber":378,"Before":null,"After":null,"FileName":false,"Score":500.14285714285717,"DebugScore":"","LineFragments":[{"LineOffset":6,"Offset":9778,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICAgIGFwaUZldGNoPEFnZW50U2Vzc2lvblN0YXRlPihgL2FwaS92MS9oYXJuZXNzLXJ1bnMvJHtlbmNvZGVVUklDb21wb25lbnQoaGFybmVzc1J1bklEKX0vYWdlbnQtc2Vzc2lvbi9zdG9wYCwgeyBtZXRob2Q6ICdQT1NUJywgdG9rZW4gfSksCg==","LineStart":9833,"LineEnd":9972,"LineNumber":379,"Before":null,"After":null,"FileName":false,"Score":500.07142857142856,"DebugScore":"","LineFragments":[{"LineOffset":13,"Offset":9846,"MatchLength":12,"SymbolInfo":null}]}],"Checksum":"osWLPhZ6HxI=","Score":5000000009.391424}
{"FileName":"internal/controlplaneapi/api.go","Repository":"008-01_zoekt-search-skill","Language":"Go","LineMatches":[{"Line":"CUFnZW50U2Vzc2lvbnMgKkFnZW50U2Vzc2lvblNlcnZpY2UK","LineStart":1003,"LineEnd":1039,"LineNumber":37,"Before":null,"After":null,"FileName":false,"Score":501,"DebugScore":"","LineFragments":[{"LineOffset":1,"Offset":1004,"MatchLength":12,"SymbolInfo":null},{"LineOffset":16,"Offset":1019,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQl9LCBmdW5jKHcgaHR0cC5SZXNwb25zZVdyaXRlciwgciAqaHR0cC5SZXF1ZXN0KSB7IGEuaGFuZGxlUnVuQWdlbnRTZXNzaW9uR2V0KHcsIHIsIGlkKSB9KQo=","LineStart":8448,"LineEnd":8540,"LineNumber":188,"Before":null,"After":null,"FileName":false,"Score":500.9761904761905,"DebugScore":"","LineFragments":[{"LineOffset":63,"Offset":8511,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQl9LCBmdW5jKHcgaHR0cC5SZXNwb25zZVdyaXRlciwgciAqaHR0cC5SZXF1ZXN0KSB7IGEuaGFuZGxlUnVuQWdlbnRTZXNzaW9uQ3JlYXRlKHcsIHIsIGlkKSB9KQo=","LineStart":8830,"LineEnd":8925,"LineNumber":194,"Before":null,"After":null,"FileName":false,"Score":500.95238095238096,"DebugScore":"","LineFragments":[{"LineOffset":63,"Offset":8893,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQl9LCBmdW5jKHcgaHR0cC5SZXNwb25zZVdyaXRlciwgciAqaHR0cC5SZXF1ZXN0KSB7IGEuaGFuZGxlUnVuQWdlbnRTZXNzaW9uUHJvbXB0KHcsIHIsIGlkKSB9KQo=","LineStart":9395,"LineEnd":9490,"LineNumber":203,"Before":null,"After":null,"FileName":false,"Score":500.92857142857144,"DebugScore":"","LineFragments":[{"LineOffset":63,"Offset":9458,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQl9LCBmdW5jKHcgaHR0cC5SZXNwb25zZVdyaXRlciwgciAqaHR0cC5SZXF1ZXN0KSB7IGEuaGFuZGxlUnVuQWdlbnRTZXNzaW9uRXZlbnRzKHcsIHIsIGlkKSB9KQo=","LineStart":9801,"LineEnd":9896,"LineNumber":209,"Before":null,"After":null,"FileName":false,"Score":500.9047619047619,"DebugScore":"","LineFragments":[{"LineOffset":63,"Offset":9864,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQl9LCBmdW5jKHcgaHR0cC5SZXNwb25zZVdyaXRlciwgciAqaHR0cC5SZXF1ZXN0KSB7IGEuaGFuZGxlUnVuQWdlbnRTZXNzaW9uRXZlbnRzU3RyZWFtKHcsIHIsIGlkKSB9KQo=","LineStart":10237,"LineEnd":10338,"LineNumber":215,"Before":null,"After":null,"FileName":false,"Score":500.8809523809524,"DebugScore":"","LineFragments":[{"LineOffset":63,"Offset":10300,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQl9LCBmdW5jKHcgaHR0cC5SZXNwb25zZVdyaXRlciwgciAqaHR0cC5SZXF1ZXN0KSB7IGEuaGFuZGxlUnVuQWdlbnRTZXNzaW9uU3RvcCh3LCByLCBpZCkgfSkK","LineStart":10647,"LineEnd":10740,"LineNumber":221,"Before":null,"After":null,"FileName":false,"Score":500.85714285714283,"DebugScore":"","LineFragments":[{"LineOffset":63,"Offset":10710,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CUFnZW50U2Vzc2lvbiAgICAgICAgICAgICpvcGVyYXRvcnYxYWxwaGExLkFnZW50U2Vzc2lvblNwZWMgYGpzb246ImFnZW50U2Vzc2lvbixvbWl0ZW1wdHkiYAo=","LineStart":21211,"LineEnd":21303,"LineNumber":453,"Before":null,"After":null,"FileName":false,"Score":500.8333333333333,"DebugScore":"","LineFragments":[{"LineOffset":1,"Offset":21212,"MatchLength":12,"SymbolInfo":null},{"LineOffset":43,"Offset":21254,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CVBoYXNlICAgICBvcGVyYXRvcnYxYWxwaGExLkFnZW50U2Vzc2lvblBoYXNlIGBqc29uOiJwaGFzZSxvbWl0ZW1wdHkiYAo=","LineStart":22419,"LineEnd":22490,"LineNumber":487,"Before":null,"After":null,"FileName":false,"Score":500.8095238095238,"DebugScore":"","LineFragments":[{"LineOffset":28,"Offset":22447,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CUFnZW50U2Vzc2lvbiAgICAgICAqYWdlbnRTZXNzaW9uUmVzcG9uc2UgICAgICAgICAgICBganNvbjoiYWdlbnRTZXNzaW9uLG9taXRlbXB0eSJgCg==","LineStart":23140,"LineEnd":23225,"LineNumber":499,"Before":null,"After":null,"FileName":false,"Score":500.7857142857143,"DebugScore":"","LineFragments":[{"LineOffset":1,"Offset":23141,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CWlmIHJ1bi5TcGVjLkFnZW50U2Vzc2lvbiAhPSBuaWwgfHwgcnVuLlN0YXR1cy5BZ2VudFNlc3Npb24gIT0gbmlsIHsK","LineStart":23841,"LineEnd":23910,"LineNumber":521,"Before":null,"After":null,"FileName":false,"Score":500.76190476190476,"DebugScore":"","LineFragments":[{"LineOffset":13,"Offset":23854,"MatchLength":12,"SymbolInfo":null},{"LineOffset":47,"Offset":23888,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQlpZiBydW4uU3BlYy5BZ2VudFNlc3Npb24gIT0gbmlsIHsK","LineStart":23951,"LineEnd":23987,"LineNumber":523,"Before":null,"After":null,"FileName":false,"Score":500.73809523809524,"DebugScore":"","LineFragments":[{"LineOffset":14,"Offset":23965,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJYWdlbnRTZXNzaW9uLlJ1bnRpbWUgPSBydW4uU3BlYy5BZ2VudFNlc3Npb24uUnVudGltZQo=","LineStart":23987,"LineEnd":24043,"LineNumber":524,"Before":null,"After":null,"FileName":false,"Score":500.7142857142857,"DebugScore":"","LineFragments":[{"LineOffset":35,"Offset":24022,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJYWdlbnRTZXNzaW9uLkFnZW50ID0gcnVuLlNwZWMuQWdlbnRTZXNzaW9uLkFnZW50Cg==","LineStart":24043,"LineEnd":24095,"LineNumber":525,"Before":null,"After":null,"FileName":false,"Score":500.6904761904762,"DebugScore":"","LineFragments":[{"LineOffset":33,"Offset":24076,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQlpZiBydW4uU3RhdHVzLkFnZW50U2Vzc2lvbiAhPSBuaWwgewo=","LineStart":24099,"LineEnd":24137,"LineNumber":527,"Before":null,"After":null,"FileName":false,"Score":500.6666666666667,"DebugScore":"","LineFragments":[{"LineOffset":16,"Offset":24115,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJaWYgcnVuLlN0YXR1cy5BZ2VudFNlc3Npb24uUnVudGltZSAhPSAiIiB7Cg==","LineStart":24137,"LineEnd":24183,"LineNumber":528,"Before":null,"After":null,"FileName":false,"Score":500.64285714285717,"DebugScore":"","LineFragments":[{"LineOffset":17,"Offset":24154,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJCWFnZW50U2Vzc2lvbi5SdW50aW1lID0gcnVuLlN0YXR1cy5BZ2VudFNlc3Npb24uUnVudGltZQo=","LineStart":24183,"LineEnd":24242,"LineNumber":529,"Before":null,"After":null,"FileName":false,"Score":500.6190476190476,"DebugScore":"","LineFragments":[{"LineOffset":38,"Offset":24221,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJaWYgcnVuLlN0YXR1cy5BZ2VudFNlc3Npb24uQWdlbnQgIT0gIiIgewo=","LineStart":24247,"LineEnd":24291,"LineNumber":531,"Before":null,"After":null,"FileName":false,"Score":500.5952380952381,"DebugScore":"","LineFragments":[{"LineOffset":17,"Offset":24264,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJCWFnZW50U2Vzc2lvbi5BZ2VudCA9IHJ1bi5TdGF0dXMuQWdlbnRTZXNzaW9uLkFnZW50Cg==","LineStart":24291,"LineEnd":24346,"LineNumber":532,"Before":null,"After":null,"FileName":false,"Score":500.57142857142856,"DebugScore":"","LineFragments":[{"LineOffset":36,"Offset":24327,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJYWdlbnRTZXNzaW9uLlNlc3Npb25JRCA9IHJ1bi5TdGF0dXMuQWdlbnRTZXNzaW9uLlNlc3Npb25JRAo=","LineStart":24351,"LineEnd":24413,"LineNumber":534,"Before":null,"After":null,"FileName":false,"Score":500.54761904761904,"DebugScore":"","LineFragments":[{"LineOffset":39,"Offset":24390,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJYWdlbnRTZXNzaW9uLlBoYXNlID0gcnVuLlN0YXR1cy5BZ2VudFNlc3Npb24uUGhhc2UK","LineStart":24413,"LineEnd":24467,"LineNumber":535,"Before":null,"After":null,"FileName":false,"Score":500.5238095238095,"DebugScore":"","LineFragments":[{"LineOffset":35,"Offset":24448,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQlBZ2VudFNlc3Npb246ICAgICAgIGFnZW50U2Vzc2lvbiwK","LineStart":24820,"LineEnd":24856,"LineNumber":547,"Before":null,"After":null,"FileName":false,"Score":500.5,"DebugScore":"","LineFragments":[{"LineOffset":2,"Offset":24822,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CWlmIHJlcS5BZ2VudFNlc3Npb24gIT0gbmlsIHsK","LineStart":26524,"LineEnd":26554,"LineNumber":601,"Before":null,"After":null,"FileName":false,"Score":500.4761904761905,"DebugScore":"","LineFragments":[{"LineOffset":8,"Offset":26532,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQlyZXEuQWdlbnRTZXNzaW9uLkFwcGx5RGVmYXVsdHMoKQo=","LineStart":26554,"LineEnd":26589,"LineNumber":602,"Before":null,"After":null,"FileName":false,"Score":500.45238095238096,"DebugScore":"","LineFragments":[{"LineOffset":6,"Offset":26560,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQlpZiBlcnIgOj0gcmVxLkFnZW50U2Vzc2lvbi5WYWxpZGF0ZSgpOyBlcnIgIT0gbmlsIHsK","LineStart":26589,"LineEnd":26643,"LineNumber":603,"Before":null,"After":null,"FileName":false,"Score":500.42857142857144,"DebugScore":"","LineFragments":[{"LineOffset":16,"Offset":26605,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CXZhciBhZ2VudFNlc3Npb25TdGF0dXMgKm9wZXJhdG9ydjFhbHBoYTEuQWdlbnRTZXNzaW9uU3RhdHVzCg==","LineStart":26870,"LineEnd":26931,"LineNumber":615,"Before":null,"After":null,"FileName":false,"Score":500.4047619047619,"DebugScore":"","LineFragments":[{"LineOffset":42,"Offset":26912,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CWlmIHJlcS5BZ2VudFNlc3Npb24gIT0gbmlsICYmIHJlcS5BZ2VudFNlc3Npb24uRW5hYmxlZCgpIHsK","LineStart":26931,"LineEnd":26991,"LineNumber":616,"Before":null,"After":null,"FileName":false,"Score":500.3809523809524,"DebugScore":"","LineFragments":[{"LineOffset":8,"Offset":26939,"MatchLength":12,"SymbolInfo":null},{"LineOffset":35,"Offset":26966,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQlhZ2VudFNlc3Npb25TdGF0dXMgPSAmb3BlcmF0b3J2MWFscGhhMS5BZ2VudFNlc3Npb25TdGF0dXN7Cg==","LineStart":26991,"LineEnd":27052,"LineNumber":617,"Before":null,"After":null,"FileName":false,"Score":500.35714285714283,"DebugScore":"","LineFragments":[{"LineOffset":41,"Offset":27032,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJUnVudGltZTogcmVxLkFnZW50U2Vzc2lvbi5SdW50aW1lLAo=","LineStart":27052,"LineEnd":27090,"LineNumber":618,"Before":null,"After":null,"FileName":false,"Score":500.3333333333333,"DebugScore":"","LineFragments":[{"LineOffset":16,"Offset":27068,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJQWdlbnQ6ICAgcmVxLkFnZW50U2Vzc2lvbi5BZ2VudCwK","LineStart":27090,"LineEnd":27126,"LineNumber":619,"Before":null,"After":null,"FileName":false,"Score":500.3095238095238,"DebugScore":"","LineFragments":[{"LineOffset":16,"Offset":27106,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJUGhhc2U6ICAgb3BlcmF0b3J2MWFscGhhMS5BZ2VudFNlc3Npb25QaGFzZVByb3Zpc2lvbmluZywK","LineStart":27126,"LineEnd":27186,"LineNumber":620,"Before":null,"After":null,"FileName":false,"Score":500.2857142857143,"DebugScore":"","LineFragments":[{"LineOffset":29,"Offset":27155,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJQWdlbnRTZXNzaW9uOiAgICAgICAgICAgIHJlcS5BZ2VudFNlc3Npb24sCg==","LineStart":27969,"LineEnd":28015,"LineNumber":641,"Before":null,"After":null,"FileName":false,"Score":500.26190476190476,"DebugScore":"","LineFragments":[{"LineOffset":3,"Offset":27972,"MatchLength":12,"SymbolInfo":null},{"LineOffset":32,"Offset":28001,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQlTdGF0dXM6IG9wZXJhdG9ydjFhbHBoYTEuSGFybmVzc1J1blN0YXR1c3tBZ2VudFNlc3Npb246IGFnZW50U2Vzc2lvblN0YXR1c30sCg==","LineStart":28127,"LineEnd":28206,"LineNumber":645,"Before":null,"After":null,"FileName":false,"Score":500.23809523809524,"DebugScore":"","LineFragments":[{"LineOffset":44,"Offset":28171,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CXZhciByZXN1bWVkQWdlbnRTZXNzaW9uICpvcGVyYXRvcnYxYWxwaGExLkFnZW50U2Vzc2lvblN0YXR1cwo=","LineStart":32533,"LineEnd":32595,"LineNumber":766,"Before":null,"After":null,"FileName":false,"Score":500.2142857142857,"DebugScore":"","LineFragments":[{"LineOffset":12,"Offset":32545,"MatchLength":12,"SymbolInfo":null},{"LineOffset":43,"Offset":32576,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CWlmIHJ1bi5TcGVjLkFnZW50U2Vzc2lvbiAhPSBuaWwgJiYgcnVuLlNwZWMuQWdlbnRTZXNzaW9uLkVuYWJsZWQoKSB7Cg==","LineStart":32595,"LineEnd":32665,"LineNumber":767,"Before":null,"After":null,"FileName":false,"Score":500.1904761904762,"DebugScore":"","LineFragments":[{"LineOffset":13,"Offset":32608,"MatchLength":12,"SymbolInfo":null},{"LineOffset":45,"Offset":32640,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQlyZXN1bWVkQWdlbnRTZXNzaW9uID0gJm9wZXJhdG9ydjFhbHBoYTEuQWdlbnRTZXNzaW9uU3RhdHVzewo=","LineStart":32665,"LineEnd":32727,"LineNumber":768,"Before":null,"After":null,"FileName":false,"Score":500.1666666666667,"DebugScore":"","LineFragments":[{"LineOffset":9,"Offset":32674,"MatchLength":12,"SymbolInfo":null},{"LineOffset":42,"Offset":32707,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJUnVudGltZTogcnVuLlNwZWMuQWdlbnRTZXNzaW9uLlJ1bnRpbWUsCg==","LineStart":32727,"LineEnd":32770,"LineNumber":769,"Before":null,"After":null,"FileName":false,"Score":500.14285714285717,"DebugScore":"","LineFragments":[{"LineOffset":21,"Offset":32748,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJQWdlbnQ6ICAgcnVuLlNwZWMuQWdlbnRTZXNzaW9uLkFnZW50LAo=","LineStart":32770,"LineEnd":32811,"LineNumber":770,"Before":null,"After":null,"FileName":false,"Score":500.1190476190476,"DebugScore":"","LineFragments":[{"LineOffset":21,"Offset":32791,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQkJUGhhc2U6ICAgb3BlcmF0b3J2MWFscGhhMS5BZ2VudFNlc3Npb25QaGFzZVByb3Zpc2lvbmluZywK","LineStart":32811,"LineEnd":32871,"LineNumber":771,"Before":null,"After":null,"FileName":false,"Score":500.0952380952381,"DebugScore":"","LineFragments":[{"LineOffset":29,"Offset":32840,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQlTdGF0dXM6ICAgICBvcGVyYXRvcnYxYWxwaGExLkhhcm5lc3NSdW5TdGF0dXN7QWdlbnRTZXNzaW9uOiByZXN1bWVkQWdlbnRTZXNzaW9ufSwK","LineStart":33187,"LineEnd":33271,"LineNumber":778,"Before":null,"After":null,"FileName":false,"Score":500.07142857142856,"DebugScore":"","LineFragments":[{"LineOffset":48,"Offset":33235,"MatchLength":12,"SymbolInfo":null},{"LineOffset":69,"Offset":33256,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQlhZ2VudFRyYW5zcG9ydCA9IG5ld1BvZFByb3h5QWdlbnRTZXNzaW9uVHJhbnNwb3J0KG5hbWVzcGFjZSwgaHR0cENsaWVudCwgYmFzZVVSTCwgIiIpCg==","LineStart":38364,"LineEnd":38452,"LineNumber":938,"Before":null,"After":null,"FileName":false,"Score":500.04761904761904,"DebugScore":"","LineFragments":[{"LineOffset":30,"Offset":38394,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CQlhcGkuQWdlbnRTZXNzaW9ucyA9IG5ld0FnZW50U2Vzc2lvblNlcnZpY2UoYWdlbnRUcmFuc3BvcnQsIG5ld0FnZW50U2Vzc2lvblN0b3JlKGFnZW50U2Vzc2lvblN0b3JlUGF0aChhdWRpdFBhdGgpKSkK","LineStart":38831,"LineEnd":38948,"LineNumber":955,"Before":null,"After":null,"FileName":false,"Score":500.0238095238095,"DebugScore":"","LineFragments":[{"LineOffset":6,"Offset":38837,"MatchLength":12,"SymbolInfo":null},{"LineOffset":25,"Offset":38856,"MatchLength":12,"SymbolInfo":null},{"LineOffset":64,"Offset":38895,"MatchLength":12,"SymbolInfo":null}]}],"Checksum":"aGaAotuUe1A=","Score":5000000007.759336}
{"FileName":"web/src/ui/pages/RunDetailPage.tsx","Repository":"008-01_zoekt-search-skill","Language":"TSX","LineMatches":[{"Line":"ICAgICgpID0+IGFwaS5nZXRBZ2VudFNlc3Npb24odG9rZW4sIGlkKSwK","LineStart":1615,"LineEnd":1657,"LineNumber":40,"Before":null,"After":null,"FileName":false,"Score":501,"DebugScore":"","LineFragments":[{"LineOffset":17,"Offset":1632,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICAgICgpID0+IGFwaS5saXN0QWdlbnRTZXNzaW9uRXZlbnRzKHRva2VuLCBpZCwgeyBsaW1pdDogMjAwIH0pLAo=","LineStart":1871,"LineEnd":1936,"LineNumber":45,"Before":null,"After":null,"FileName":false,"Score":500.90909090909093,"DebugScore":"","LineFragments":[{"LineOffset":18,"Offset":1889,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICBjb25zdCBzdGFydEFnZW50U2Vzc2lvbiA9IHVzZUNhbGxiYWNrKGFzeW5jICgpID0+IHsK","LineStart":3193,"LineEnd":3247,"LineNumber":84,"Before":null,"After":null,"FileName":false,"Score":500.8181818181818,"DebugScore":"","LineFragments":[{"LineOffset":13,"Offset":3206,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICAgICAgYXdhaXQgYXBpLmNyZWF0ZUFnZW50U2Vzc2lvbih0b2tlbiwgaWQpCg==","LineStart":3304,"LineEnd":3350,"LineNumber":88,"Before":null,"After":null,"FileName":false,"Score":500.72727272727275,"DebugScore":"","LineFragments":[{"LineOffset":22,"Offset":3326,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICBjb25zdCBwcm9tcHRBZ2VudFNlc3Npb24gPSB1c2VDYWxsYmFjayhhc3luYyAoKSA9PiB7Cg==","LineStart":3664,"LineEnd":3719,"LineNumber":99,"Before":null,"After":null,"FileName":false,"Score":500.6363636363636,"DebugScore":"","LineFragments":[{"LineOffset":14,"Offset":3678,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICAgICAgYXdhaXQgYXBpLnByb21wdEFnZW50U2Vzc2lvbih0b2tlbiwgaWQsIGFnZW50UHJvbXB0LnRyaW0oKSkK","LineStart":3818,"LineEnd":3884,"LineNumber":104,"Before":null,"After":null,"FileName":false,"Score":500.54545454545456,"DebugScore":"","LineFragments":[{"LineOffset":22,"Offset":3840,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICBjb25zdCBzdG9wQWdlbnRTZXNzaW9uID0gdXNlQ2FsbGJhY2soYXN5bmMgKCkgPT4gewo=","LineStart":4236,"LineEnd":4289,"LineNumber":116,"Before":null,"After":null,"FileName":false,"Score":500.45454545454544,"DebugScore":"","LineFragments":[{"LineOffset":12,"Offset":4248,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICAgICAgYXdhaXQgYXBpLnN0b3BBZ2VudFNlc3Npb24odG9rZW4sIGlkKQo=","LineStart":4346,"LineEnd":4390,"LineNumber":120,"Before":null,"After":null,"FileName":false,"Score":500.3636363636364,"DebugScore":"","LineFragments":[{"LineOffset":20,"Offset":4366,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICAgICAgICAgICAgICA8QnRuIGRpc2FibGVkPXthZ2VudEFjdGluZyB8fCB0b2tlbi50cmltKCkgPT09ICIifSBvbkNsaWNrPXtzdGFydEFnZW50U2Vzc2lvbn0gdHlwZT0iYnV0dG9uIj4K","LineStart":8593,"LineEnd":8701,"LineNumber":208,"Before":null,"After":null,"FileName":false,"Score":500.27272727272725,"DebugScore":"","LineFragments":[{"LineOffset":79,"Offset":8672,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICAgICAgICAgICAgICA8QnRuIHZhcmlhbnQ9ImRhbmdlciIgZGlzYWJsZWQ9e2FnZW50QWN0aW5nIHx8IHRva2VuLnRyaW0oKSA9PT0gIiJ9IG9uQ2xpY2s9e3N0b3BBZ2VudFNlc3Npb259IHR5cGU9ImJ1dHRvbiI+Cg==","LineStart":8800,"LineEnd":8924,"LineNumber":211,"Before":null,"After":null,"FileName":false,"Score":500.1818181818182,"DebugScore":"","LineFragments":[{"LineOffset":95,"Offset":8895,"MatchLength":12,"SymbolInfo":null}]},{"Line":"ICAgICAgICAgICAgICA8QnRuIHZhcmlhbnQ9InByaW1hcnkiIGRpc2FibGVkPXthZ2VudEFjdGluZyB8fCB0b2tlbi50cmltKCkgPT09ICIiIHx8IGFnZW50UHJvbXB0LnRyaW0oKSA9PT0gIiJ9IG9uQ2xpY2s9e3Byb21wdEFnZW50U2Vzc2lvbn0gdHlwZT0iYnV0dG9uIj4K","LineStart":9291,"LineEnd":9447,"LineNumber":219,"Before":null,"After":null,"FileName":false,"Score":500.09090909090907,"DebugScore":"","LineFragments":[{"LineOffset":127,"Offset":9418,"MatchLength":12,"SymbolInfo":null}]}],"Checksum":"bXmjaXZNZAU=","Score":5000000006.81881}
{"FileName":"internal/controlplanecli/client.go","Repository":"008-01_zoekt-search-skill","Language":"Go","LineMatches":[{"Line":"dHlwZSBBZ2VudFNlc3Npb25JbmZvIHN0cnVjdCB7Cg==","LineStart":465,"LineEnd":496,"LineNumber":25,"Before":null,"After":null,"FileName":false,"Score":501,"DebugScore":"","LineFragments":[{"LineOffset":5,"Offset":470,"MatchLength":12,"SymbolInfo":null}]},{"Line":"CUFnZW50U2Vzc2lvbiAgICAgICAqQWdlbnRTZXNzaW9uSW5mbyBganNvbjoiYWdlbnRTZXNzaW9uLG9taXRlbXB0eSJgCg==","LineStart":1203,"LineEnd":1273,"LineNumber":41,"Before":null,"After":null,"FileName":false,"Score":500.5,"DebugScore":"","LineFragments":[{"LineOffset":1,"Offset":1204,"MatchLength":12,"SymbolInfo":null},{"LineOffset":21,"Offset":1224,"MatchLength":12,"SymbolInfo":null}]}],"Checksum":"eaSmRS+Bz5U=","Score":5000000006.832642}
{"FileName":".agents/skills/zoekt-search/SKILL.md","Repository":"008-01_zoekt-search-skill","Language":"Markdown","LineMatches":[{"Line":"YmFzaCAuYWdlbnRzL3NraWxscy96b2VrdC1zZWFyY2gvc2NyaXB0cy96b2VrdC1zZWFyY2guc2ggIkFnZW50U2Vzc2lvbiIK","LineStart":1158,"LineEnd":1230,"LineNumber":28,"Before":null,"After":null,"FileName":false,"Score":501,"DebugScore":"","LineFragments":[{"LineOffset":58,"Offset":1216,"MatchLength":12,"SymbolInfo":null}]},{"Line":"YmFzaCAuYWdlbnRzL3NraWxscy96b2VrdC1zZWFyY2gvc2NyaXB0cy96b2VrdC1zZWFyY2guc2ggIkFnZW50U2Vzc2lvbiIK","LineStart":1337,"LineEnd":1409,"LineNumber":37,"Before":null,"After":null,"FileName":false,"Score":500.75,"DebugScore":"","LineFragments":[{"LineOffset":58,"Offset":1395,"MatchLength":12,"SymbolInfo":null}]},{"Line":"YmFzaCAuYWdlbnRzL3NraWxscy96b2VrdC1zZWFyY2gvc2NyaXB0cy96b2VrdC1zZWFyY2guc2ggInR5cGUgQWdlbnRTZXNzaW9uIHN0cnVjdCIK","LineStart":1525,"LineEnd":1609,"LineNumber":43,"Before":null,"After":null,"FileName":false,"Score":500.5,"DebugScore":"","LineFragments":[{"LineOffset":63,"Offset":1588,"MatchLength":12,"SymbolInfo":null}]},{"Line":"YmFzaCAuYWdlbnRzL3NraWxscy96b2VrdC1zZWFyY2gvc2NyaXB0cy96b2VrdC1zZWFyY2guc2ggLS1uby1qc29uICJBZ2VudFNlc3Npb24iCg==","LineStart":1960,"LineEnd":2042,"LineNumber":55,"Before":null,"After":null,"FileName":false,"Score":500.25,"DebugScore":"","LineFragments":[{"LineOffset":68,"Offset":2028,"MatchLength":12,"SymbolInfo":null}]}],"Checksum":"nzUS9htpOlA=","Score":5000000006.459198}
```

The JSONL output contains base64-encoded line content (for binary safety), plus plain-text FileName and LineNumber fields. Results are ranked by relevance — 5 files matched across Go, TypeScript, TSX, and Markdown.

## Step 4: Search for "func.*Start" — regex results

Zoekt supports full regex. Using --no-json for human-readable output.

```bash
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh --no-json "func.*Start" 2>/dev/null | head -30
```

```output
.agents/skills/zoekt-search/SKILL.md:40:bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "func.*Start"
internal/controlplanecli/agent_start.go:24:func runAgentStartCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
.agents/skills/zoekt-search/scripts/test-integration.sh:56:func StartAgent(name string) *AgentSession {
.agents/skills/zoekt-search/scripts/test-integration.sh:177:# --- Test 3: Search for regex pattern (func.*Start) -------------------------
.agents/skills/zoekt-search/scripts/test-integration.sh:179:printf '\n--- Test 3: Search for "func.*Start" (regex) ---\n'
.agents/skills/zoekt-search/scripts/test-integration.sh:181:regex_output=$(bash "${SCRIPT_DIR}/zoekt-search.sh" --index-dir "${INDEX_DIR}" "func.*Start" 2>/dev/null) || true
.agents/skills/zoekt-search/scripts/test-integration.sh:184:regex_plain=$(bash "${SCRIPT_DIR}/zoekt-search.sh" --index-dir "${INDEX_DIR}" --no-json "func.*Start" 2>/dev/null) || true
.agents/skills/zoekt-search/scripts/test-integration.sh:187:    pass "Regex search 'func.*Start' found StartAgent (plain text)"
.agents/skills/zoekt-search/scripts/test-integration.sh:189:    fail "Regex search 'func.*Start' did not find StartAgent"
internal/controlplanecli/agent_session.go:248:func (c *Client) StartAgent(ctx context.Context, workspaceID, repoURL, repoRevision, agent, image string, imagePullSecrets []string, egressMode string) (runID string, err error) {
internal/controlplanecli/agent_start_test.go:13:func TestAgentStart_Success(t *testing.T) {
internal/controlplanecli/agent_start_test.go:110:func TestAgentStart_MissingRequiredFlags(t *testing.T) {
internal/controlplanecli/agent_start_test.go:150:func TestAgentStart_Timeout(t *testing.T) {
internal/controlplanecli/agent_start_test.go:213:func TestAgentStart_ReuseWorkspace(t *testing.T) {
internal/controlplanecli/agent_start_test.go:287:func TestAgentStart_JSONOutput(t *testing.T) {
internal/controlplanecli/agent_session_test.go:280:func TestStartAgent(t *testing.T) {
internal/controlplanecli/agent_session_test.go:359:func TestStartAgentWithExistingWorkspace(t *testing.T) {
internal/controlplanecli/agent_integration_test.go:424:func TestFullLifecycle_StartExecStop(t *testing.T) {
internal/controlplanecli/agent_integration_test.go:548:func TestFullLifecycle_StartListStatus(t *testing.T) {
web/src/ui/pages/SessionsPage.tsx:44:function formatStarted(raw: string | undefined): { age: string; exact: string; title: string } {
internal/symphony/runner/appserver_test.go:115:func TestRunStartupTimeout(t *testing.T) {
```

## Step 5: Search for something that doesn't exist

Searching for a nonsense string returns empty output.

```bash
result=$(bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "xyzzy_nonexistent_pattern_42" 2>/dev/null); if [ -z "$result" ]; then echo "(no results)"; else echo "$result"; fi
```

```output
(no results)
```

## Step 6: Search with --no-json — plain text output

The --no-json flag produces human-readable output with file paths, line numbers, and matched content.

```bash
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh --no-json "type.*struct" 2>/dev/null | head -20
```

```output
internal/config/config.go:12:type Runtime struct {
internal/auditlog/store.go:17:type Event struct {
internal/auditlog/store.go:30:type Store struct {
.agents/skills/zoekt-search/SKILL.md:43:bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "type AgentSession struct"
internal/controlplaneapi/api.go:28:type API struct {
internal/controlplaneapi/api.go:42:type Options struct {
internal/controlplaneapi/api.go:324:type sessionCreateRequest struct {
internal/controlplaneapi/api.go:329:type sessionResponse struct {
internal/controlplaneapi/api.go:443:type runCreateRequest struct {
internal/controlplaneapi/api.go:483:type agentSessionResponse struct {
internal/controlplaneapi/api.go:490:type runResponse struct {
internal/controlplaneapi/api.go:788:type attachControlRequest struct {
internal/controlplaneapi/api.go:820:type egressOverrideRequest struct {
internal/controlplaneapi/json.go:13:type httpError struct {
internal/controlplaneapi/json.go:28:type requestError struct {
internal/controlplaneapi/auth.go:22:type Authenticator struct {
internal/controlplanecli/status.go:10:type SessionStatus struct {
internal/controlplaneapi/tokens.go:13:type TokenRecord struct {
internal/controlplaneapi/tokens.go:21:type TokenStore struct {
internal/controlplanecli/errors.go:14:type APIError struct {
```

## Step 7: OpenCode plugin for auto-reindexing

The zoekt-search skill includes an OpenCode plugin that watches for file changes and reindexes automatically. Here is the skill structure:

```bash
find .agents/skills/zoekt-search -type f | sort
```

```output
.agents/skills/zoekt-search/bin/.gitkeep
.agents/skills/zoekt-search/bin/zoekt
.agents/skills/zoekt-search/bin/zoekt-index
.agents/skills/zoekt-search/scripts/install-zoekt.sh
.agents/skills/zoekt-search/scripts/test-integration.sh
.agents/skills/zoekt-search/scripts/zoekt-index.sh
.agents/skills/zoekt-search/scripts/zoekt-search.sh
.agents/skills/zoekt-search/SKILL.md
```

## Step 8: Integration test

Run the full integration test suite to prove everything works end-to-end.

```bash
bash .agents/skills/zoekt-search/scripts/test-integration.sh 2>&1
```

```output

--- Test 1: Index temp directory ---
PASS: zoekt-index.sh completed successfully
PASS: Index contains 1 shard file(s)

--- Test 2: Search for "AgentSession" (JSONL) ---
PASS: Search for 'AgentSession' returned non-empty results
PASS: JSONL results include main.go
PASS: JSONL results include session.ts
PASS: Decoded JSONL Line content contains 'AgentSession'

--- Test 3: Search for "func.*Start" (regex) ---
PASS: Regex search 'func.*Start' found StartAgent (plain text)
PASS: Regex JSONL results include main.go

--- Test 4: Search for non-existent pattern ---
PASS: Search for non-existent pattern returned empty results

--- Test 5: Search with --no-json ---
PASS: --no-json search returned results containing handleRequest
PASS: --no-json output is not JSON formatted

--- Test 6: Verify JSONL output format ---
PASS: JSONL output is valid JSON

=== Results ===
Passed: 12
Failed: 0

All tests passed.
```
