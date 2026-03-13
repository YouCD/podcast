package byte_dance

import (
	"context"
	"testing"

	"podcast/config"
)

func TestParseScript(t *testing.T) {
	a := `[S1] 老王老王老王！紧急呼叫！
[S2] 嚯，这什么动静，我刚泡的咖啡差点洒我芯片上。
[S1] 别提芯片了！我正刷手机呢，就感觉这世界…怎么说呢，一边是疯了一样的涨价，一边是神仙打架，还有一边眼看着要炸了！
[S2] 你这个描述…精准中带着点抽象。来来来，坐下慢慢喷，是不是苹果又提价了？
[S1] 何止是提价啊！那新出的什么MacBook Neo，键盘灯都没了，触控板跟砖头似的，它还敢涨价？！这跟抢劫有啥区别？还有，我早上用GitHub卡了半天，结果你猜怎么着？有人要给它来个“平替”！
[S2] 你这一下子就踩中了今天俩最大的雷。一个叫“苹果教你重新定义性价比”，一个叫“OpenAI想当开发者的新房东”。
[S1] 对对对！还有那个什么阿里，闹得跟宫廷剧似的，技术大牛跑了。再有就是……哦对了，我感觉我看的很多开源项目都跟垃圾场似的，全是AI在那儿瞎写。诶，那边中东好像又不太平，黄金涨了。

`
	err := PodCast(context.Background(), config.Cfg.ByteDance.AppID, config.Cfg.ByteDance.AccessToken, "/home/ycd/self_data/source_code/podcast/pkg/protocols/podcast_final.mp3", a)
	if err != nil {
		t.Error(err)
	}
}
