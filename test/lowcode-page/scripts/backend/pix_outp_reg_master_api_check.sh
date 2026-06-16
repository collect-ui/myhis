#!/usr/bin/env bash
set -u
BASE_URL="${PIX_OUTP_BASE_URL:-http://127.0.0.1:8017}"
API_URL="${PIX_OUTP_API_URL:-$BASE_URL/template_data/data}"
OUTPUT_DIR="${PIX_OUTP_OUTPUT_DIR:-$(pwd)}"
TS="$(date +%Y%m%d%H%M%S)"
mkdir -p "$OUTPUT_DIR"
BACKEND_LOG="$OUTPUT_DIR/backend.log"
exec > >(tee "$BACKEND_LOG") 2>&1

check() {
  local label="$1" cond="$2"
  echo "$([ "$cond" = "1" ] && echo "✅" || echo "❌") $label"
}

echo "=== 门诊号池 CRUD 验证 ==="

api() {
  local svc="$1"; shift
  curl --noproxy '*' -sS -m 60 -X POST "$API_URL" -F "service=$svc" "$@"
}

extract() {
  python3 -c "
import sys,json,os
raw=sys.stdin.read()
decoder=json.JSONDecoder()
pos=0
while pos<len(raw):
  try:
    d,idx=decoder.raw_decode(raw,pos)
    if d.get('success'):
      sys.stdout.write(json.dumps(d,ensure_ascii=False))
      sys.stdout.flush()
      os._exit(0)
    pos=idx
  except:pos+=1
sys.stdout.write('{\"success\":false}')
sys.stdout.flush()
os._exit(0)
"
}

# 1. 分页查询
r1=$(api 'him.pix_outp_reg_master_query' -F 'page=1' -F 'size=3')
s1=$(echo "$r1" | extract | python3 -c "import sys,json;d=json.load(sys.stdin);print(int(d.get('count',0)>0 and len(d.get('data',[]))==3))" 2>/dev/null || echo "0")
check "分页查询" "$s1"

# 2. 主键过滤
r2=$(echo "$r1" | extract | python3 -c "import sys,json;d=json.load(sys.stdin);print(d['data'][0]['master_id'])" 2>/dev/null || echo "")
r3=$(api 'him.pix_outp_reg_master_query' -F "master_id=$r2" -F 'pagination=false')
s2=$(echo "$r3" | extract | python3 -c "import sys,json;d=json.load(sys.stdin);print(int(len(d.get('data',[]))==1))" 2>/dev/null || echo "0")
check "主键过滤" "$s2"

# 3. 新增
MID="$(date +%s%N | cut -c1-12)"
r4=$(api 'him.pix_outp_reg_master_save' \
  -F "master_id=$MID" -F 'area_code=001' -F 'doctor_code=DOC001' \
  -F 'outp_type_code=T1' -F 'is_enable=1' -F 'master_status=0' \
  -F 'sort_no=1' -F 'registration_limits=50' -F 'alias_name=测试号源')
s3=$(echo "$r4" | extract | python3 -c "import sys,json;print(int(json.load(sys.stdin).get('success',False)))" 2>/dev/null || echo "0")
check "新增" "$s3"

# 4. 修改
r5=$(api 'him.pix_outp_reg_master_update' \
  -F "master_id=$MID" -F 'alias_name=更新测试' -F 'registration_limits=100')
s4=$(echo "$r5" | extract | python3 -c "import sys,json;print(int(json.load(sys.stdin).get('success',False)))" 2>/dev/null || echo "0")
check "修改" "$s4"

# 5. 验证修改
r6=$(api 'him.pix_outp_reg_master_query' -F "master_id=$MID" -F 'pagination=false')
s5=$(echo "$r6" | extract | python3 -c "
import sys,json
d=json.load(sys.stdin)
if d.get('success') and len(d.get('data',[]))>0:
  r=d['data'][0]
  print(int(r.get('alias_name')=='更新测试' and str(r.get('registration_limits'))=='100'))
else: print(0)" 2>/dev/null || echo "0")
check "验证修改" "$s5"

# 6. 删除
r7=$(api 'him.pix_outp_reg_master_delete' -F "master_id_list=$MID")
s6=$(echo "$r7" | extract | python3 -c "import sys,json;print(int(json.load(sys.stdin).get('success',False)))" 2>/dev/null || echo "0")
check "删除" "$s6"

# 7. 验证删除
r8=$(api 'him.pix_outp_reg_master_query' -F "master_id=$MID" -F 'pagination=false')
s7=$(echo "$r8" | extract | python3 -c "import sys,json;d=json.load(sys.stdin);print(int(d.get('success',False) and len(d.get('data',[]))==0))" 2>/dev/null || echo "0")
check "验证删除" "$s7"

P=$(python3 -c "print($s1+$s2+$s3+$s4+$s5+$s6+$s7)" 2>/dev/null || echo "0")
echo ""
echo "$P / 7 项通过"
[ "$P" -lt 7 ] && exit 1 || echo "✅ 全部通过"
