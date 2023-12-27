const { createRoot } = ReactDOM
const { useEffect, useState, useRef } = React

let imgName = ''

const App = () => {

    const [msg, setmsg] = useState([])
    const [value, setvalue] = useState('')
    const [linkShow, setlinkShow] = useState("")
    const [pullStart, setpullStart] = useState(false)
    const msgDom = useRef()
    const eventSource = useRef()
    const loading = useRef(false)

    useEffect(() => {
        msgDom.current.scrollTop = msgDom.current.scrollHeight
    }, [msg])


    const onerror = (e) => {
        console.log(e);
        eventSource.current.close()
        eventSource.current.removeEventListener("start", start)
        eventSource.current.removeEventListener("live", live)
        eventSource.current.removeEventListener("info", info)
        eventSource.current.removeEventListener("sus", sus)
        eventSource.current.removeEventListener("err", err)
        loading.current = false
    }

    const live = () => {
        console.log('sse live');
    }

    const start = (e) => {
        setmsg((prev) => prev.concat({
            time: dayjs().format('YY-MM-DD HH:mm:ss'),
            data: "连接成功"
        }))
    }

    const info = (e) => {
        setmsg((prev) => prev.concat({
            time: dayjs().format('YY-MM-DD HH:mm:ss'),
            data: e.data
        }))
    }

    const sus = (e) => {
        setmsg((prev) => prev.concat({
            time: dayjs().format('YY-MM-DD HH:mm:ss'),
            data: "拉取成功，开始下载..."
        }))
        loading.current = false
        setlinkShow(e.data)
        const a = document.createElement('a')
        a.href = e.data
        a.download = imgName + '.tar'
        a.click()
    }

    const err = (e) => {
        setmsg((prev) => prev.concat({
            time: dayjs().format('YY-MM-DD HH:mm:ss'),
            data: e.data
        }))
        loading.current = false
    }

    const pull = () => {
        setmsg([])
        if (value === '') {
            alert("参数错误")
            return
        }
        const valueArr = value.split(/\s+/)
        if (valueArr.length !== 3) {
            alert("参数错误")
            return
        }
        let [img, tag = "latest"] = valueArr[2].split(':')
        imgName = img.includes("/") ? img.split("/")[0] : img
        tag = encodeURIComponent(tag)
        img = encodeURIComponent(img)
        if (loading.current) return
        loading.current = true
        setpullStart(true)
        eventSource.current = new EventSource(`/api/pull?img=${img}&tag=${tag}`)
        eventSource.current.addEventListener("start", start)
        eventSource.current.addEventListener("live", live)
        eventSource.current.addEventListener("info", info)
        eventSource.current.addEventListener("sus", sus)
        eventSource.current.addEventListener("err", err)
        eventSource.current.onerror = onerror
    }

    return <div className="page">
        <div className="input">
            <input className="input-dom" placeholder="docker pull busybox" value={value}
                onChange={(e) => setvalue(e.target.value)} />
            <span className="input-btn" onClick={pull}></span>
        </div>
        <div className="use" style={{ display: pullStart ? "none" : "block" }}>
            <p>前往 <a target="__blank" href="https://hub.docker.com/">https://hub.docker.com/</a> 搜索镜像</p>
            <p>输入拉取命令，点击图标</p>
            <p>等待拉取完成，自动下载</p>
        </div>
        <div className="info" style={{ height: pullStart ? '50%' : '0' }}>
            <div className="msg" ref={msgDom}>
                {msg.map((item, index) => <p className="msg-item" key={index}>
                    <span className="msg-time">{item.time}</span>
                    {item.data}
                </p>)}
                <p className="msg-item" style={{ visibility: "hidden" }}>-</p>
            </div>
        </div>
        <div style={{ visibility: linkShow != "" ? "visible" : "hidden" }} className="link">
            下载地址:
            <a className="link-a" href={linkShow} target="__blank">
                {window.location.origin + linkShow}
            </a>
        </div>
    </div>
}

const container = document.getElementById('app');
const root = createRoot(container);
root.render(<App />);
