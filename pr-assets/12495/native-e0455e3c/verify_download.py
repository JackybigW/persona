"""Offline gallery verifier. Use --all-originals when all 37 original ZIPs are next to the boards."""
from pathlib import Path
import argparse, hashlib, json, struct, zipfile, re, sys
from urllib.parse import urlsplit,unquote

def verify(root, all_originals=False):
    m=json.loads((root/'manifest.json').read_text()); pages=m['pages']; checked={'boards':0,'sourceArchives':0,'originalPNGs':0,'originalAX':0,'bytesBoards':0,'bytesOriginals':0,'markdownLinks':0}
    def check(condition,msg):
        if not condition:raise ValueError(msg)
    def sha(b):return hashlib.sha256(b).hexdigest()
    def dims(b):check(b[:8]==b'\x89PNG\r\n\x1a\n','invalid PNG');return list(struct.unpack('>II',b[16:24]))
    ids=[p['id'] for p in pages];check(len(ids)==len(set(ids))==37,'page coverage')
    for f in sorted(root.rglob('*.md')):
        for target in re.findall(r'\]\(([^)]+)\)',f.read_text()):
            u=urlsplit(target)
            if u.scheme:continue
            dest=(f.parent/unquote(u.path)).resolve()
            check(root.resolve() in dest.parents,'link escaping site '+target)
            if dest.name=='offline-review.zip' or (not all_originals and dest.parent.name=='originals'):continue
            check(dest.is_file(),'missing link '+target);checked['markdownLinks']+=1
    for p in pages:
        archive=root/p['originals']['file']
        expected={}
        for mode,frame in p['modes'].items():
            board=frame['board'];b=(root/board['file']).read_bytes()
            check(sha(b)==board['sha256'] and dims(b)==board['dimensions'] and len(b)<=500*1024,'board '+p['id']+'/'+mode)
            checked['boards']+=1;checked['bytesBoards']+=len(b)
            check(set(frame['devices'])=={'iphone','matched','standard'},'missing device')
            for dev,r in frame['devices'].items():
                expected[r['pngEntry']]=(r['pngSHA256'],True,r['pixels'])
                expected[r['axEntry']]=(r['axSHA256'],False,[])
                x,y,w,h=board['crops'][dev];bw,bh=board['dimensions']
                check(x>=0 and y>=0 and x+w<=bw and y+h<=bh and abs(w/h-r['pixels'][0]/r['pixels'][1])<.001,'invalid native aspect crop')
        if not archive.exists() and not all_originals:continue
        check(sha(archive.read_bytes())==p['originals']['sha256'],'source archive '+p['id'])
        checked['sourceArchives']+=1;checked['bytesOriginals']+=archive.stat().st_size
        with zipfile.ZipFile(archive) as z:
            check(set(z.namelist())==set(expected) and len(z.namelist())==18,'raw file inventory')
            for name,(digest,image,pixels) in expected.items():
                dat=z.read(name);check(sha(dat)==digest,'raw file '+p['id']+'/'+name)
                if image:check(dims(dat)==pixels,'source dimensions');checked['originalPNGs']+=1
                else:
                    check(len(dat)>80,'empty hierarchy')
                    if name.endswith('.json'):check(len(json.loads(dat))>5,'empty HarmonyOS tree')
                    else:check(b'Application' in dat or b'Window' in dat,'empty XCTest tree')
                    checked['originalAX']+=1
    check(checked['boards']==111 and checked['bytesBoards']<=52*1024**2,'board total or budget')
    if all_originals:check(checked['sourceArchives']==37 and checked['originalPNGs']==checked['originalAX']==333,'source coverage')
    paired=m['pairedWorktrees'];check(len(paired['cycles'])==2 and paired['samePairingRestored'],'worktrees')
    for cycle in paired['cycles']:
        steps=cycle['steps']
        check(steps['baseline']['mainHead']==steps['cleanup-verified']['mainHead']==cycle['baselineHEAD'],'main HEAD')
        check(steps['dirty-preview']['changes']==1 and steps['dirty-cancel-verified']['fixtureFileIntact'] and steps['cleanup-verified']['branchRemoved'] and steps['cleanup-verified']['hostWorkspaceRemoved'] and steps['cleanup-verified']['folderRemoved'],'worktree result')
    return checked
if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--all-originals',action='store_true');parser.add_argument('--root',type=Path,default=Path(__file__).resolve().parent);args=parser.parse_args()
    print(json.dumps(verify(args.root,args.all_originals),indent=2))
