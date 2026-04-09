const handlerChangeShownColumn = (el) => {
    let key = el.id.split("-")[1]

    let minColumn = document.querySelector("#mac_min-" + key)
    let maxColumn = document.querySelector("#mac_max-" + key)

    if (!minColumn || !maxColumn) return

    if (minColumn.style.display === "none") {
        minColumn.style.display = "table-cell"
        maxColumn.style.display = "none"
    } else {
        maxColumn.style.display = "table-cell"
        minColumn.style.display = "none"
    }
}

function handlerAddEltexSwitches() {
    const textarea = document.querySelector("#ips")
    if (!textarea) return

    fetch("/snmp/eltex/switches", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded; charset=UTF-8"
        },
        body: new URLSearchParams({
            ips: textarea.value
        })
    })
        .then(response => response.json())
        .then(data => {
            if (data && data.ok) {
                window.location.reload()
                return
            }
            alert("не удалось добавить IP")
        })
        .catch(error => {
            console.error(error)
            alert("не удалось добавить IP")
        })
}

function handlerDeleteEltexSwitch(ip) {
    fetch("/snmp/eltex/switches/delete", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded; charset=UTF-8"
        },
        body: new URLSearchParams({
            ip: ip
        })
    })
        .then(response => response.json())
        .then(data => {
            if (data && data.ok) {
                window.location.reload()
                return
            }
            alert("не удалось удалить IP")
        })
        .catch(error => {
            console.error(error)
            alert("не удалось удалить IP")
        })
}

function handlerTransformColumn(el) {
    let col = el.parentNode
    let value = col.innerText
    let btn = document.createElement("button")
    let key = col.id.split("-")[1]

    col.innerHTML = `<input type='text' value="${value}" id="input-${key}"/> `

    btn.setAttribute("onclick", `handlerSendChange(${key})`)
    btn.innerHTML = "Сохранить"
    col.append(btn)
}

function handlerTransformBandwidthColumn(el) {
    let col = el.parentNode
    let key = col.id.split("-")[1]

    let text = col.innerText || ""

    // Default values: "NoLimit" -> 0, otherwise integer.
    let tx = 0
    let rx = 0

    let txMatch = text.match(/TX:([^\s]+)/i)
    if (txMatch && txMatch[1] && txMatch[1] !== "NoLimit") {
        let n = Number(txMatch[1])
        if (Number.isFinite(n)) tx = n
    }

    let rxMatch = text.match(/RX:([^\s]+)/i)
    if (rxMatch && rxMatch[1] && rxMatch[1] !== "NoLimit") {
        let n = Number(rxMatch[1])
        if (Number.isFinite(n)) rx = n
    }

    col.innerHTML = `
        <input type='number' value='${tx}' id='bandwidth-tx-${key}' min='0'/>
        <input type='number' value='${rx}' id='bandwidth-rx-${key}' min='0'/>
    `

    let btn = document.createElement("button")
    btn.setAttribute("onclick", `handlerSendBandwidthChange(${key})`)
    btn.innerHTML = "Сохранить"
    col.append(btn)
}

function handlerGetTransceiverInfo(key) {
    let ip = window.location.href.split("snmp/eltex/")[1]

    let options = {
        method: "POST",
        body: JSON.stringify(key)
    }

    fetch(`/snmp/eltex/${ip}/transceiver-info`, options)
        .then(response => response.json())
        .then(data => {
            let transceiverColumn = document.querySelector(`#transceiver_min-${key}`)
            if (!transceiverColumn) return
            transceiverColumn.innerHTML = `<span>tx: ${data.TransceiverTransmission}</span> <span>rx: ${data.TransceiverReception}</span>`
        })
        .catch(error => console.error(error))
}

function handlerSendChange(key) {
    let value = document.querySelector(`#input-${key}`).value
    let switchModel = document.querySelector("h1").innerHTML.match(/\(([^)]+)\)/)[1]

    let ip

    if (window.location.href.includes("dlink")) {
        ip = window.location.href.split("snmp/dlink/")[1]
    } else if (window.location.href.includes("eltex")) {
        ip = window.location.href.split("snmp/eltex/")[1]
    }

    let options = {
        method: "POST",
        body: JSON.stringify({
            Index: Number(key),
            Description: value,
            SwitchModel: switchModel
        })
    }

    fetch("/snmp/dlink/change_port_description/" + ip, options)
        .then(response => response.json())
        .then(data => {
            if (data && data.ok) {
                let col = document.querySelector(`#description-${key}`)
                col.innerHTML = `${value} <img onclick="handlerTransformColumn(this)" src="/snmp/assets/public/pen.svg" alt="O">`
            } else {
                alert("не удалось изменить описание")
            }
        })
        .catch(error => console.error(error))
}

function handlerSendBandwidthChange(key) {
    let switchModel = document.querySelector("h1").innerHTML.match(/\(([^)]+)\)/)[1]
    let ip = window.location.href.split("snmp/dlink/")[1]

    let tx = Number(document.querySelector(`#bandwidth-tx-${key}`).value)
    let rx = Number(document.querySelector(`#bandwidth-rx-${key}`).value)

    if (!Number.isFinite(tx) || tx < 0 || !Number.isFinite(rx) || rx < 0) {
        alert("Bandwidth должен быть integer >= 0 (0 = NoLimit)")
        return
    }

    let options = {
        method: "POST",
        body: JSON.stringify({
            Index: Number(key),
            SwitchModel: switchModel,
            BandwidthTX: Math.trunc(tx),
            BandwidthRX: Math.trunc(rx)
        })
    }

    fetch("/snmp/dlink/change_bandwidth/" + ip, options)
        .then(response => response.json())
        .then(data => {
            if (data && data.ok) {
                let col = document.querySelector(`#bandwidth-${key}`)

                let txText = (tx === 0) ? "NoLimit" : String(Math.trunc(tx))
                let rxText = (rx === 0) ? "NoLimit" : String(Math.trunc(rx))

                col.innerHTML = `TX:${txText} RX:${rxText} <img onclick=\"handlerTransformBandwidthColumn(this)\" src=\"/snmp/assets/public/pen.svg\" alt=\"O\">`
            } else {
                alert("не удалось изменить bandwidth")
            }
        })
        .catch(error => console.error(error))
}

function loadMacs() {
    let ip
    let url

    if (window.location.href.includes("/snmp/dlink/")) {
        ip = window.location.href.split("snmp/dlink/")[1]
        url = `/snmp/dlink/${ip}/macs`
    } else if (window.location.href.includes("/snmp/eltex/")) {
        ip = window.location.href.split("snmp/eltex/")[1]
        url = `/snmp/eltex/${ip}/macs`
    } else {
        return
    }

    // Initialize all rows to 0 while loading.
    document.querySelectorAll("td[id^='mac_min-']").forEach(td => {
        td.classList.add("loading")
    })

    fetch(url)
        .then(r => r.json())
        .then(data => {
            if (!data || !data.ok || !data.ports) {
                throw new Error("bad response")
            }

            // Fill defaults for all visible rows.
            document.querySelectorAll("td[id^='mac_min-']").forEach(td => {
                const key = td.id.split("-")[1]
                const maxTd = document.querySelector(`#mac_max-${key}`)
                td.classList.remove("loading")
                td.onclick = () => handlerChangeShownColumn(td)
                td.innerHTML = `<span class="amount">0</span> <img src="/snmp/assets/public/open.svg" alt="O">`
                if (maxTd) {
                    maxTd.onclick = () => handlerChangeShownColumn(maxTd)
                    maxTd.innerHTML = ``
                }
            })

            // Apply actual macs from server.
            Object.keys(data.ports).forEach(k => {
                const key = String(k)
                const macs = data.ports[key] || []

                const minTd = document.querySelector(`#mac_min-${key}`)
                const maxTd = document.querySelector(`#mac_max-${key}`)

                if (minTd) {
                    minTd.classList.remove("loading")
                    minTd.onclick = () => handlerChangeShownColumn(minTd)
                    minTd.innerHTML = `<span class="amount">${macs.length}</span> <img src="/snmp/assets/public/open.svg" alt="O">`
                }

                if (maxTd) {
                    maxTd.onclick = () => handlerChangeShownColumn(maxTd)
                    maxTd.innerHTML = macs.map(m => `<div>${m}</div>`).join("")
                }
            })
        })
        .catch(err => {
            console.error(err)
            document.querySelectorAll("td[id^='mac_min-']").forEach(td => {
                td.classList.remove("loading")
                td.innerHTML = `!`
            })
        })
}

window.addEventListener("DOMContentLoaded", loadMacs)
