#!/bin/bash
# 创建20本测试书籍

#  ┌─────┬────────────────────┬──────────┬──────────────┐
#  │  #  │        书名        │   类型   │     价格     │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 1   │ 星河剑典           │ 星际玄幻 │ vip1         │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 2   │ 霸道总裁的契约甜妻 │ 都市言情 │ 免费         │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 3   │ 末日：我有亿万物资 │ 末世重生 │ vip2 / ¥29.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 4   │ 全职高手之巅峰荣耀 │ 电子竞技 │ 免费         │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 5   │ 鬼吹灯之精绝古城   │ 盗墓探险 │ vip1 / ¥19.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 6   │ 庆余年             │ 穿越权谋 │ vip1 / ¥39.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 7   │ 完美世界           │ 东方玄幻 │ vip2 / ¥49.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 8   │ 三体               │ 科幻     │ 免费         │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 9   │ 大奉打更人         │ 穿越悬疑 │ vip1 / ¥29.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 10  │ 择天记             │ 东方玄幻 │ vip1 / ¥29.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 11  │ 斗破苍穹           │ 东方玄幻 │ vip2 / ¥59.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 12  │ 武动乾坤           │ 东方玄幻 │ vip2 / ¥59.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 13  │ 琅琊榜             │ 古装权谋 │ 免费         │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 14  │ 元龙               │ 异世玄幻 │ vip1 / ¥19.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 15  │ 凡人修仙传         │ 修仙     │ vip2 / ¥69.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 16  │ 我欲封天           │ 东方仙侠 │ vip2 / ¥59.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 17  │ 雪中悍刀行         │ 武侠     │ vip1 / ¥49.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 18  │ 大主宰             │ 东方玄幻 │ vip2 / ¥59.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 19  │ 神墓               │ 东方玄幻 │ vip1 / ¥39.9 │
#  ├─────┼────────────────────┼──────────┼──────────────┤
#  │ 20  │ 盘龙               │ 西方玄幻 │ vip1 / ¥39.9 │
#  └─────┴────────────────────┴──────────┴──────────────┘

BASE_URL="http://localhost:9929/api/v1/books/create"
CONTENT_TYPE="Content-Type: application/json"

echo "===== 开始创建20本书籍 ====="

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"星河剑典","description":"宇宙纪元2045年，人类踏入星际时代。少年林逸意外获得上古剑修传承，在星辰大海中开辟出一条独属于他的剑道。","cover_image":"https://cdn.example.com/covers/xingjianjiandian.jpg","vip_level":"vip1","price":0}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"霸道总裁的契约甜妻","description":"她是被家族抛弃的落魄千金，一场商业联姻让她嫁给了权倾一时的冷面总裁。婚后相处，他发现她与想象中完全不同，而她也在不知不觉中沦陷。","cover_image":"https://cdn.example.com/covers/tianqilaopo.jpg","vip_level":"free","price":0}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"末日：我有亿万物资","description":"丧尸爆发，全球沦陷。主角重生回到灾难前一个月，携带着亿万物资和先知优势，在废土之上重建人类文明。","cover_image":"https://cdn.example.com/covers/mori_wuzi.jpg","vip_level":"vip2","price":29.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"全职高手之巅峰荣耀","description":"电竞天才叶修被俱乐部驱逐后，在网吧重新起步，凭借对游戏的热爱和超群的实力，再次站上荣耀之巅。","cover_image":"https://cdn.example.com/covers/quanzhi.jpg","vip_level":"free","price":0}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"鬼吹灯之精绝古城","description":"胡八一与王胖子在一次偶然的探险中发现了精绝古城的遗址，由此踏上了一条充满惊险与谜团的寻宝之路。","cover_image":"https://cdn.example.com/covers/gcds.jpg","vip_level":"vip1","price":19.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"庆余年","description":"一个年轻的穿越者范闲，在庆国跌宕起伏的一生。文能提笔安天下，武能马上定乾坤，他在权谋斗争中一步步走向巅峰。","cover_image":"https://cdn.example.com/covers/qingyunian.jpg","vip_level":"vip1","price":39.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"完美世界","description":"少年石昊，天生至尊骨，却被族人挖去，沦为弃子。他以不屈意志逆天改命，在完美世界中杀出一条血路，终成一代荒天帝。","cover_image":"https://cdn.example.com/covers/wanmei.jpg","vip_level":"vip2","price":49.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"三体","description":"地球文明与三体文明的首次接触，揭开了宇宙黑暗森林法则的一角。叶文洁的一次抉择，改变了整个人类文明的命运。","cover_image":"https://cdn.example.com/covers/santi.jpg","vip_level":"free","price":0}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"大奉打更人","description":"警校生许七安意外穿越到异世界，成为大奉朝的一名打更人。他用现代刑侦知识破解奇案，在妖魔横行的世界里守护人间正义。","cover_image":"https://cdn.example.com/covers/genggeng.jpg","vip_level":"vip1","price":29.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"择天记","description":"少年陈长生为了逆天改命，离开小地方来到神都，在各大势力的博弈中艰难前行。他以无双智计和坚定道心，一步一步走向长生之路。","cover_image":"https://cdn.example.com/covers/zetianji.jpg","vip_level":"vip1","price":29.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"斗破苍穹","description":"天才少年萧炎沦为废材，受尽嘲讽与屈辱。绝望之际，戒指中的药老苏醒，传授绝世功法。萧炎以惊人毅力重返巅峰，踏上斗气大陆的传奇之路。","cover_image":"https://cdn.example.com/covers/doupocangqiong.jpg","vip_level":"vip2","price":59.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"武动乾坤","description":"炎城少年林动，为复仇而修炼，偶得祖符传承，在波澜壮阔的乱世中崛起。他与应欢欢、绫清竹的刻骨铭心之情，也令人动容。","cover_image":"https://cdn.example.com/covers/wudong.jpg","vip_level":"vip2","price":59.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"琅琊榜","description":"十二年前，赤焰军少帅林殊背负冤屈侥幸生还，更名梅长苏。他以病弱之躯重返京城，以病弱之躯搅动风云，终为七万忠魂洗清冤屈。","cover_image":"https://cdn.example.com/covers/langyabang.jpg","vip_level":"free","price":0}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"元龙","description":"特种兵王宋命穿越到元魂世界，前世的身手和战术思想在这个世界大放异彩。他以现代军事思维碾压各路天才，缔造了一个新的传奇。","cover_image":"https://cdn.example.com/covers/yuanlong.jpg","vip_level":"vip1","price":19.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"凡人修仙传","description":"资质平庸的韩立，偶得神秘小瓶瓶，在修仙界艰难求生。他没有惊才绝艳的天赋，只有脚踏实地的心性，一步一步在修仙世界中走出自己的道路。","cover_image":"https://cdn.example.com/covers/fanren.jpg","vip_level":"vip2","price":69.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"我欲封天","description":"我若要有，天不可无！我若要无，天不可有！孟浩一介凡尘书生，在修真界步步为营，以符箓之术名震天下，终成一代封天之主。","cover_image":"https://cdn.example.com/covers/woyufengtian.jpg","vip_level":"vip2","price":59.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"雪中悍刀行","description":"北凉世子徐凤年，三年游历归来，看尽世间冷暖。他从纨绔子弟成长为一代名将，在江湖与庙堂之间纵横捭阖，为天下苍生守护北凉三十万铁骑。","cover_image":"https://cdn.example.com/covers/xuezhong.jpg","vip_level":"vip1","price":49.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"大主宰","description":"大千世界，万道争锋。牧尘与洛璃携手，从北灵境走出，一步步登上大千世界的巅峰，揭开域外邪族入侵的惊天秘密。","cover_image":"https://cdn.example.com/covers/dazuizai.jpg","vip_level":"vip2","price":59.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"神墓","description":"洪荒时代，诸神黄昏。一个万年前就已存在的强者辰南，在神墓中苏醒，为探寻真相和复活爱人而踏上逆天之路。","cover_image":"https://cdn.example.com/covers/shenmu.jpg","vip_level":"vip1","price":39.9}'
echo ""

curl -s "$BASE_URL" -X POST -H "$CONTENT_TYPE" \
  -d '{"name":"盘龙","description":"小小家族的少爷林雷，无意中捡到一枚盘龙戒指，从此开启了他的传奇人生。以地系为主，融合风系和火系，一步步成为玉兰大陆最强者。","cover_image":"https://cdn.example.com/covers/panlong.jpg","vip_level":"vip1","price":39.9}'
echo ""

echo "===== 20本书籍创建完成 ====="
