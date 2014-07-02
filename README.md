# music api server

音乐服务的api通用接口，目前已支持的音乐服务:

* 虾米
    * 专辑
    * 歌曲列表
    * 精选集
* 网易云音乐
    * 专辑
    * 歌曲列表
    * 歌单

## 安装(debian/ubuntu)

首先检查`GOPATH`变量是否正确设置，如果未设置，参考[这里](http://blog.atime.me/note/golang-summary.html#3867e350ebb33a487c4ac5f7787e1c29)进行设置。

    # install redis-server
    sudo apt-get install redis-server

    # install redis driver for golang
    go get github.com/garyburd/redigo/redis

    # install music api server
    go get github.com/mawenbao/music-api-server

    # install init script
    sudo cp $GOPATH/src/github.com/mawenbao/music-api-server/tools/init-script /etc/init.d/music-api-server

    # set your GOPATH in init script
    sudo sed -i "s|/SET/YOUR/GOPATH/HERE|`echo $GOPATH`|" /etc/init.d/music-api-server

    sudo chmod +x /etc/init.d/music-api-server

    # start music api server
    sudo service music-api-server start

    # (optional) let music api server start on boot
    sudo update-rc.d music-api-server defaults

    # (optional) install logrotate config
    sudo cp $GOPATH/src/github.com/mawenbao/music-api-server/tools/logrotate-config /etc/logrotate.d/music-api-server

## 更新(debian/ubuntu)

    # update and restart music-api-server
    go get -u github.com/mawenbao/music-api-server
    sudo service music-api-server restart

    # flush redis cache
    redis-cli
    > flushall

## API
### Demo

[http://app.atime.me/music-api-server/?p=xiami&t=songlist&i=20526,1772292423&c=abc123](http://app.atime.me/music-api-server/?p=xiami&t=songlist&i=20526,1772292423&c=abc123)

### 请求

    GET http://localhost:9099/?p=xiami&t=songlist&i=20526,1772292423&c=abc123

* `localhost:9099`: 默认监听地址
* `p=xiami`: 音乐API提供商，目前支持:
    * 虾米(xiami)
    * 网易云音乐(netease)
* `t=songlist`: 音乐类型
    * songlist(xiami + netease): 歌曲列表，对应的id是半角逗号分隔的多个歌曲id
    * album(xiami + netease): 音乐专辑，对应的id为专辑id
    * collect(xiami): 虾米的精选集，对应的id为精选集id
    * playlist(netease): 网易云音乐的歌单，对应的id为歌单(playlist)id
* `i=20526,1772292423`: 歌曲/专辑/精选集/歌单的id，歌曲列表类型可用半角逗号分割多个歌曲id
* `c=abc123`: 使用jsonp方式返回数据，实际返回为`abc123({songs: ...});`

### 返回

    {
        "status": "返回状态，ok为正常，failed表示出错并设置msg",
        "msg":    "如果status为failed，这里会保存错误信息，否则不返回该字段",
        "songs": [
            {
                "name":     "歌曲名称",
                "url":      "歌曲播放地址",
                "artists":  "演唱者",
                "provider": "音乐提供商"
                "lrc_url":  "歌词文件地址(可能没有)",
            }
        ]
    } 

如果有`c=abc123`请求参数，则实际以[jsonp](http://en.wikipedia.org/wiki/JSONP)方式返回数据。

## TODO
1. 更好的缓存策略

## Thanks
* [Wordpress Hermit Player](http://mufeng.me/hermit-for-wordpress.html)
* [网易云音乐API分析](https://github.com/yanunon/NeteaseCloudMusic/wiki/%E7%BD%91%E6%98%93%E4%BA%91%E9%9F%B3%E4%B9%90API%E5%88%86%E6%9E%90)

