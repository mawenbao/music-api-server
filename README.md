# music api server

音乐服务的api通用接口，目前已支持的音乐服务只有虾米:

* 虾米
* 网易云音乐(计划中)

## 安装(debian/ubuntu)

首先检查`GOPATH`变量是否正确设置，如果未设置，参考[这里](http://blog.atime.me/note/golang-summary.html#3867e350ebb33a487c4ac5f7787e1c29)进行设置。

    # install redis-server
    sudo apt-get install redis-server

    # install redis driver for golang
    go get github.com/garyburd/redigo/redis

    # install music api server
    go get github.com/mawenbao/music-api-server

    # install init script
    sudo cp $GOPATH/src/github.com/mawenbao/music-api-server/tools/music-api-server-init-script /etc/init.d/music-api-server

    # set your GOPATH in init script
    sudo sed -i "s|/SET/YOUR/GOPATH/HERE|`echo $GOPATH`|" /etc/init.d/music-api-server

    sudo chmod +x /etc/init.d/music-api-server

    # start music api server
    sudo service music-api-server start

    # (optional) let music api server start on boot
    sudo update-rc.d music-api-server defaults

    # (optional) install logrotate config
    sudo cp $GOPATH/src/github.com/mawenbao/music-api-server/tools/logrotate-config /etc/logrotate.d/music-api-server

## API

### 请求

    GET http://localhost:9099/?p=xiami&t=songlist&i=20526,1772292423&c=abc123

* `localhost:9099`: 默认监听地址
* `p=xiami`: 音乐API提供商，目前仅支持虾米(xiami):
    * 虾米(xiami)
* `t=songlist`: 音乐类型
    * songlist(xiami): 虾米的歌曲列表，对应的id是半角逗号分隔的多个歌曲id
    * collect(xiami): 虾米的精选集，对应的id为精选集id
    * album(xiami): 虾米的专辑，对应的id为专辑id
* `i=20526,1772292423`: 歌曲/专辑/精选集的id，歌曲列表类型可用半角逗号分割多个歌曲id
* `c=abc123`: 使用jsonp方式返回数据，实际返回为`abc123({songs: ...});`

### 返回

    {
        "status": "返回状态，ok为正常，failed表示出错并设置msg",
        "msg": "如果status为failed，这里会保存错误信息，否则不返回该字段",
        "songs": [
            {
                "song_title": "歌曲名称",
                "song_src": "歌曲播放地址",
                "song_lrc": "歌词文件地址",
                "song_author": "演唱者",
                "song_provider": "音乐提供商"
            }
        ]
    }   

如果有`c=abc123`请求参数，则实际以[jsonp](http://en.wikipedia.org/wiki/JSONP)方式返回数据。

## TODO

1. 更好的缓存策略
    1. 优化songlist的缓存请求
2. 实现网易云音乐的api

