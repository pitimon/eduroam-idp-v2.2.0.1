# ระบบความปลอดภัยและการป้องกัน Brute Force

## ภาพรวมระบบความปลอดภัย

eduroam API Web Service มีระบบความปลอดภัยหลายชั้นที่ป้องกันการโจมตีแบบ brute force และการใช้รหัสยืนยันที่หมดอายุ

### การป้องกัน Brute Force

1. **Rate Limiting**
   - จำกัดจำนวนครั้งที่สามารถพยายามล็อกอินผิดพลาดได้ (ค่าเริ่มต้น: 5 ครั้ง)
   - หากเกินขีดจำกัด จะถูกล็อคเป็นเวลา 30 นาที
   - การวัดจำนวนครั้งทำโดยใช้ Redis เพื่อความเร็วและความแม่นยำ

2. **รหัสยืนยัน 8 หลัก**
   - รหัสยืนยัน 8 หลักสร้างความยากในการเดารหัส (โอกาส 1 ใน 100,000,000)
   - มีการสร้างรหัสด้วยฟังก์ชัน Cryptographic secure random

3. **จำกัดการพยายามใช้รหัสเดียวกัน**
   - อนุญาตให้ใช้รหัสยืนยันได้เพียง 5 ครั้ง
   - หากเกินจำนวนครั้ง จะต้องขอรหัสใหม่

### การจัดการรหัสที่หมดอายุ

1. **อายุรหัสยืนยัน**
   - รหัสยืนยันมีอายุ 15 นาที
   - เมื่อหมดอายุ จะไม่สามารถใช้รหัสนั้นได้อีก

2. **การยกเลิกรหัสเมื่อมีการสร้างรหัสใหม่**
   - เมื่อมีการสร้างรหัสใหม่ รหัสเก่าจะถูกยกเลิกโดยอัตโนมัติ
   - ป้องกันการใช้รหัสเก่าที่อาจรั่วไหล

3. **การยกเลิกรหัสเมื่อยืนยันสำเร็จ**
   - รหัสจะถูกยกเลิกทันทีหลังจากใช้ยืนยันสำเร็จ
   - ป้องกันการใช้รหัสซ้ำ

### การจัดการ Token

1. **Token Blacklist**
   - Token ที่ถูก logout จะถูกเพิ่มลงใน blacklist ใน Redis
   - Token ที่อยู่ใน blacklist จะไม่สามารถใช้งานได้แม้ยังไม่หมดอายุ

2. **การตรวจสอบ Token ทุกครั้งที่เรียกใช้ API**
   - ตรวจสอบความถูกต้องของ Token
   - ตรวจสอบว่า Token อยู่ใน blacklist หรือไม่
   - ตรวจสอบสิทธิ์การเข้าถึงตามโดเมนที่ได้รับอนุญาต

3. **การล้างข้อมูล Token ที่หมดอายุ**
   - มีการล้าง Token ที่หมดอายุอัตโนมัติทุกวัน
   - ป้องกันการสะสมข้อมูลที่ไม่จำเป็น

## การตั้งค่าและการปรับแต่ง

ค่าต่างๆ สามารถปรับแต่งได้ผ่านตัวแปรสภาพแวดล้อม (.env):
Rate Limiting
RATE_LIMIT_MAX_ATTEMPTS=5   # จำนวนครั้งสูงสุดที่สามารถพยายามได้
RATE_LIMIT_WINDOW=3600      # ช่วงเวลาที่วัดจำนวนครั้งที่พยายาม (วินาที)
LOCKOUT_DURATION=1800       # ระยะเวลาที่ถูกล็อค (วินาที)

## การแก้ไขปัญหา

กรณีผู้ใช้ถูกล็อคเนื่องจากพยายามเข้าสู่ระบบมากเกินไป:

1. **ล้างการล็อคด้วย Redis CLI**:
redis-cli DEL "auth:attempts:email:user@example.com"

1. **ล้าง Token จาก Blacklist**:
redis-cli DEL "auth:blacklist:eyJhbGciOiJIUzI1..."

## การตรวจสอบความปลอดภัย

1. **ตรวจสอบความพยายามเข้าสู่ระบบ**:
```bash
# ดูจำนวนครั้งที่พยายามเข้าสู่ระบบของผู้ใช้
redis-cli HGETALL "auth:attempts:email:user@example.com"

ตรวจสอบ Token ที่ถูก Blacklist:
bash# ดูรายการ Token ที่ถูก Blacklist
redis-cli KEYS "auth:blacklist:*"


EOL
17. สร้างสคริปต์สำหรับตรวจสอบความพยายาม brute force
cat > security-monitoring.sh << 'EOL'
#!/bin/bash
สีสำหรับแสดงผล
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color
echo -e "YELLOW=====BruteForceAttemptMonitor====={YELLOW}===== Brute Force Attempt Monitor =====
YELLOW=====BruteForceAttemptMonitor====={NC}"

ตรวจสอบความพยายาม brute force จาก Redis
echo -e "\nYELLOWCheckingloginattempts:{YELLOW}Checking login attempts:
YELLOWCheckingloginattempts:{NC}"
ATTEMPT_KEYS=$(redis-cli KEYS "auth:attempts:*")

if [ -z "$ATTEMPT_KEYS" ]; then
    echo -e "GREENNologinattemptsfound.{GREEN}No login attempts found.
GREENNologinattemptsfound.{NC}"
else
    echo -e "YELLOWFoundloginattempts:{YELLOW}Found login attempts:
YELLOWFoundloginattempts:{NC}"
    for KEY in $ATTEMPT_KEYS; do
        ATTEMPTS=(redis−cliHGET"(redis-cli HGET "
(redis−cliHGET"KEY" "count")
        LAST_ATTEMPT=(redis−cliHGET"(redis-cli HGET "
(redis−cliHGET"KEY" "last_attempt")
        LAST_TIME=(date−d@"(date -d @"
(date−d@"LAST_ATTEMPT" "+%Y-%m-%d %H:%M:%S")

    IDENTIFIER=${KEY#auth:attempts:}
    
    if [ "$ATTEMPTS" -ge 5 ]; then
        echo -e "${RED}[$IDENTIFIER] $ATTEMPTS attempts. Last: $LAST_TIME${NC}"
    else
        echo -e "[$IDENTIFIER] $ATTEMPTS attempts. Last: $LAST_TIME"
    fi
done
fi
ตรวจสอบ token ที่ถูก blacklist
echo -e "\nYELLOWCheckingblacklistedtokens:{YELLOW}Checking blacklisted tokens:
YELLOWCheckingblacklistedtokens:{NC}"
BLACKLIST_KEYS=$(redis-cli KEYS "auth:blacklist:*")

if [ -z "$BLACKLIST_KEYS" ]; then
    echo -e "GREENNoblacklistedtokensfound.{GREEN}No blacklisted tokens found.
GREENNoblacklistedtokensfound.{NC}"
else
    echo -e "YELLOWFoundblacklistedtokens:{YELLOW}Found blacklisted tokens:
YELLOWFoundblacklistedtokens:{NC}"
    for KEY in $BLACKLIST_KEYS; do
        TTL=(redis−cliTTL"(redis-cli TTL "
(redis−cliTTL"KEY")
        if [ "$TTL" -gt 0 ]; then
            EXPIRY_TIME=(date−d"+(date -d "+
(date−d"+TTL seconds" "+%Y-%m-%d %H:%M:%S")
            echo "Token will expire at: EXPIRYTIME(EXPIRY_TIME (
EXPIRYT​IME({TTL}s)"
        else
            echo -e "REDTokenexpiredbutstillinblacklist.{RED}Token expired but still in blacklist.
REDTokenexpiredbutstillinblacklist.{NC}"
        fi
    done
fi

ตรวจสอบ user attempts จากฐานข้อมูล
echo -e "\nYELLOWRecentverificationattempts:{YELLOW}Recent verification attempts:
YELLOWRecentverificationattempts:{NC}"
PGPASSWORD=${POSTGRES_PASSWORD:-eduroampass} psql -h ${POSTGRES_HOST:-postgres} -U ${POSTGRES_USER:-eduroam} -d ${POSTGRES_DB:-eduroam_db} -c "
SELECT
    u.email,
    vc.purpose,
    COUNT(*) as attempts,
    MAX(vc.created_at) as last_attempt
FROM
    verification_codes vc
JOIN
    users u ON vc.user_id = u.id
WHERE
    vc.created_at > NOW() - INTERVAL '24 hours'
GROUP BY
    u.email, vc.purpose
ORDER BY
    last_attempt DESC
LIMIT 10;
"

echo -e "\nYELLOWRecentloginactivity:{YELLOW}Recent login activity:
YELLOWRecentloginactivity:{NC}"
PGPASSWORD=${POSTGRES_PASSWORD:-eduroampass} psql -h ${POSTGRES_HOST:-postgres} -U ${POSTGRES_USER:-eduroam} -d ${POSTGRES_DB:-eduroam_db} -c "
SELECT
    u.email,
    al.action,
    al.ip_address,
    al.created_at
FROM
    access_log al
JOIN
    users u ON al.user_id = u.id
WHERE
    al.created_at > NOW() - INTERVAL '24 hours'
ORDER BY
    al.created_at DESC
LIMIT 10;
"

echo -e "\nGREENMonitoringcomplete.{GREEN}Monitoring complete.
GREENMonitoringcomplete.{NC}"
EOL
chmod +x security-monitoring.sh

18. สร้างคำสั่งสำหรับปลดล็อคผู้ใช้ที่ถูกล็อค
cat > unlock-user.sh << 'EOL'
#!/bin/bash
สีสำหรับแสดงผล
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color
if [ $# -ne 1 ]; then
echo -e "${RED}Usage: $0 <email>${NC}"
exit 1
fi
EMAIL=$1
EMAIL_KEY="auth:attempts:email:$EMAIL"
IP_KEY="auth:attempts:ip:*"
echo -e "{YELLOW}Checking if user $EMAIL is locked...
{NC}"

ตรวจสอบความพยายามเข้าสู่ระบบจากอีเมล
ATTEMPTS=(redis−cliHGET"(redis-cli HGET "
(redis−cliHGET"EMAIL_KEY" "count")

if [ -z "ATTEMPTS"]∣∣["ATTEMPTS" ] || [ "
ATTEMPTS"]∣∣["ATTEMPTS" -lt 5 ]; then
    echo -e "{GREEN}User $EMAIL is not locked by email.
{NC}"
else
    echo -e "{RED}User $EMAIL is locked with $ATTEMPTS attempts.
{NC}"

# ปลดล็อคโดยลบ key
redis-cli DEL "$EMAIL_KEY"
echo -e "${GREEN}User $EMAIL has been unlocked.${NC}"
fi
ตรวจสอบความพยายามเข้าสู่ระบบจาก IP
IP_KEYS=(redis−cliKEYS"(redis-cli KEYS "
(redis−cliKEYS"IP_KEY")

if [ -z "$IP_KEYS" ]; then
    echo -e "GREENNoIPaddressesarelocked.{GREEN}No IP addresses are locked.
GREENNoIPaddressesarelocked.{NC}"
else
    echo -e "YELLOWCheckingIPlocks:{YELLOW}Checking IP locks:
YELLOWCheckingIPlocks:{NC}"
    for KEY in $IP_KEYS; do
        ATTEMPTS=(redis−cliHGET"(redis-cli HGET "
(redis−cliHGET"KEY" "count")

    if [ "$ATTEMPTS" -ge 5 ]; then
        echo -e "${RED}IP $KEY is locked with $ATTEMPTS attempts.${NC}"
        echo -e "${YELLOW}Do you want to unlock this IP? (y/n)${NC}"
        read answer
        
        if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
            redis-cli DEL "$KEY"
            echo -e "${GREEN}IP $KEY has been unlocked.${NC}"
        else
            echo -e "${YELLOW}IP $KEY remains locked.${NC}"
        fi
    fi
done
fi
ยกเลิกรหัสที่หมดอายุสำหรับผู้ใช้
echo -e "\n{YELLOW}Invalidating expired verification codes for $EMAIL...
{NC}"
PGPASSWORD=${POSTGRES_PASSWORD:-eduroampass} psql -h ${POSTGRES_HOST:-postgres} -U ${POSTGRES_USER:-eduroam} -d ${POSTGRES_DB:-eduroam_db} -c "
UPDATE verification_codes
SET expires_at = NOW()
WHERE user_id = (SELECT id FROM users WHERE email = '$EMAIL')
  AND expires_at > NOW();
"
echo -e "GREENExpiredverificationcodeshavebeeninvalidated.{GREEN}Expired verification codes have been invalidated.
GREENExpiredverificationcodeshavebeeninvalidated.{NC}"

echo -e "\nGREENUnlockprocesscompleted.{GREEN}Unlock process completed.
GREENUnlockprocesscompleted.{NC}"
EOL
chmod +x unlock-user.sh


ชุดคำสั่งนี้ได้เพิ่มระบบจัดการรหัสที่หมดอายุและป้องกันการทดลองรหัสซ้ำๆ (brute force) อย่างครบถ้วน โดยมีฟีเจอร์ดังนี้:

1. **Rate Limiting**: จำกัดจำนวนครั้งที่พยายามล็อกอินผิดพลาด โดยใช้ Redis เป็นฐานข้อมูลชั่วคราว
2. **การป้องกัน Brute Force**: ล็อคบัญชีหรือ IP ที่พยายามเข้าระบบมากเกินไป
3. **การจัดการรหัสที่หมดอายุ**: รหัสยืนยัน 8 หลักจะหมดอายุใน 15 นาที
4. **Token Blacklist**: ป้องกันการใช้ token ที่ถูกยกเลิกหรือ logout แล้ว
5. **การติดตามความพยายามเข้าระบบ**: ระบบบันทึกความพยายามล็อกอินทั้งหมด
6. **คำสั่งสำหรับการแก้ไขปัญหา**: สคริปต์สำหรับปลดล็อคผู้ใช้และตรวจสอบความพยายาม brute force

โค้ดเหล่านี้สร้าง middleware ที่จำเป็นและให้บริการสำหรับการตรวจสอบและจัดการรหัสยืนยัน 8 หลัก พร้อมระบบป้องกันการทดลองรหัสซ้ำๆ อย่างครบถ้วนRetryClaude does not have internet access. Links provided may not be accurate or up to date.